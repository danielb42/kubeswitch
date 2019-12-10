package main

import (
	"bufio"
	"log"
	"os"
	"sort"

	"github.com/gdamore/tcell"
	"github.com/rivo/tview"

	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"

	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
	_ "k8s.io/client-go/plugin/pkg/client/auth/openstack"
)

type referenceHelper struct {
	cluster   string
	namespace string
	user      string
}

var (
	kubeconfLocation = os.Getenv("HOME") + "/.kube/kubeswitch.yaml"
	namespacesFile   = os.Getenv("HOME") + "/.kubeswitch_namespaces"
	config           *clientcmdapi.Config
)

func main() {
	var err error

	if len(os.Args) > 1 && os.Args[1] == "--init" {
		createConfig()
		os.Exit(0)
	}

	loadingRules := &clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeconfLocation}
	config, err = loadingRules.Load()
	if err != nil {
		log.Fatalln(err)
	}

	app := tview.NewApplication()
	nodeRoot := tview.NewTreeNode("⛅").SetSelectable(false)
	highlightNode := nodeRoot

	for _, clusterName := range mapKeysToSortedArray(config.Clusters) {
		nodeClusterName := tview.NewTreeNode(" " + clusterName).SetColor(tcell.ColorGreen).SetSelectable(false)
		nodeRoot.AddChild(nodeClusterName)

		for _, namespace := range readUsersNamespaces() {
			nodeNamespace := tview.NewTreeNode(" " + namespace).SetReference(referenceHelper{clusterName, namespace, "user-" + namespace + "-" + clusterName})
			nodeClusterName.AddChild(nodeNamespace)
			nodeNamespace.SetSelectedFunc(func() {
				app.Stop()
				doSwitch(nodeNamespace.GetReference().(referenceHelper))
			})

			if _, ok := config.Contexts["kubeswitch"]; ok {
				if clusterName == config.Contexts["kubeswitch"].Cluster &&
					namespace == config.Contexts["kubeswitch"].Namespace {
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
	config.Contexts["kubeswitch"] = &clientcmdapi.Context{
		LocationOfOrigin: kubeconfLocation,
		Cluster:          rh.cluster,
		Namespace:        rh.namespace,
		AuthInfo:         rh.user,
	}

	config.CurrentContext = "kubeswitch"

	configAccess := clientcmd.NewDefaultClientConfigLoadingRules()
	if err := clientcmd.ModifyConfig(configAccess, *config, false); err != nil {
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

	if len(namespaces) == 0 {
		log.Fatal("could not read any namespaces from " + namespacesFile)
	}

	return namespaces
}

func createConfig() {
	if len(os.Args) < 3 {
		log.Fatal("ERROR: no kubeconfig locations specified")
	}

	loadingRules := &clientcmd.ClientConfigLoadingRules{Precedence: os.Args[1:]}
	mergedConfig, err := loadingRules.Load()
	if err != nil {
		log.Fatal(err)
	}

	if err := clientcmd.WriteToFile(*mergedConfig, kubeconfLocation); err != nil {
		log.Fatal(err)
	}

	log.Println("merged kubeconfig written to " + kubeconfLocation)
}

func mapKeysToSortedArray(m map[string]*clientcmdapi.Cluster) []string {
	var s []string

	for k := range m {
		s = append(s, k)
	}

	sort.Strings(s)
	return s
}
