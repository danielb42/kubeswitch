package main

import (
	"bufio"
	"log"
	"os"
	"sort"

	"github.com/gdamore/tcell"
	"github.com/rivo/tview"

	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

type referenceHelper struct {
	cluster   string
	namespace string
	user      string
}

var (
	kubeconfLocation = os.Getenv("HOME") + "/.kube/kubeswitch.yaml"
	namespacesFile   = os.Getenv("HOME") + "/.kubeswitch_namespaces"
	mergedConfig     *clientcmdapi.Config
)

func main() {
	var err error

	loadingRules := &clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeconfLocation}
	mergedConfig, err = loadingRules.Load()
	if err != nil {
		log.Fatalln(err)
	}

	app := tview.NewApplication()
	nodeRoot := tview.NewTreeNode("⛅").SetSelectable(false)
	highlightNode := nodeRoot

	for _, clusterName := range mapKeysToSortedArray(mergedConfig.Clusters) {
		nodeClusterName := tview.NewTreeNode(" " + clusterName).SetColor(tcell.ColorGreen).SetSelectable(false)
		nodeRoot.AddChild(nodeClusterName)

		for _, namespace := range readUsersNamespaces() {
			nodeNamespace := tview.NewTreeNode(" " + namespace).SetReference(referenceHelper{clusterName, namespace, "user-" + clusterName})
			nodeClusterName.AddChild(nodeNamespace)
			nodeNamespace.SetSelectedFunc(func() {
				app.Stop()
				doSwitch(nodeNamespace.GetReference().(referenceHelper))
			})

			if _, ok := mergedConfig.Contexts["kubeswitch"]; ok {
				if clusterName == mergedConfig.Contexts["kubeswitch"].Cluster &&
					namespace == mergedConfig.Contexts["kubeswitch"].Namespace {
					nodeNamespace.SetColor(tcell.ColorGreen)
					highlightNode = nodeNamespace
				}
			}
		}
	}

	tree := tview.NewTreeView().
		SetRoot(nodeRoot).
		SetCurrentNode(highlightNode)

	if err := app.SetRoot(tree, true).Run(); err != nil {
		log.Fatalln(err)
	}
}

func doSwitch(rh referenceHelper) {
	mergedConfig.Contexts["kubeswitch"] = &clientcmdapi.Context{
		LocationOfOrigin: kubeconfLocation,
		Cluster:          rh.cluster,
		Namespace:        rh.namespace,
		AuthInfo:         rh.user,
	}

	mergedConfig.CurrentContext = "kubeswitch"

	configAccess := clientcmd.NewDefaultClientConfigLoadingRules()
	if err := clientcmd.ModifyConfig(configAccess, *mergedConfig, false); err != nil {
		log.Fatalln(err)
	}

	log.Printf("switched to %s/%s", rh.cluster, rh.namespace)
}

func readUsersNamespaces() []string {
	file, err := os.Open(namespacesFile)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	var namespaces []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		namespaces = append(namespaces, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
	return namespaces
}

func mapKeysToSortedArray(m map[string]*clientcmdapi.Cluster) []string {
	var s []string

	for k := range m {
		s = append(s, k)
	}

	sort.Strings(s)
	return s
}
