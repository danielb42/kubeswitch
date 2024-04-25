package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubernetes "k8s.io/client-go/kubernetes"
	clientcmd "k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"

	_ "k8s.io/client-go/plugin/pkg/client/auth/azure"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
)

type contextNamespaceTuple struct {
	k8sContext   string
	k8sNamespace string
}

var (
	kubeconfLocation = os.Getenv("HOME") + "/.kube/config"
	mergedConfig     *clientcmdapi.Config
)

func main() {
	var err error

	if len(os.Args) > 1 {
		if os.Args[1] == "-h" || os.Args[1] == "--help" {
			printUsage()
		}
	}

	if len(os.Getenv("KUBECONFIG")) > 0 {
		kubeconfLocation = os.Getenv("KUBECONFIG")
	}

	loadingRules := &clientcmd.ClientConfigLoadingRules{Precedence: strings.Split(kubeconfLocation, ":")}

	mergedConfig, err = loadingRules.Load()
	if err != nil {
		log.Fatalln(err)
	}

	if len(os.Args) > 1 {
		quickSwitch()
	}

	app := tview.NewApplication()

	nodeRoot := tview.NewTreeNode("Contexts").SetSelectable(false)

	expandedNode := new(tview.TreeNode)
	highlightNode := nodeRoot

	for _, thisContextName := range mapKeysToSortedArray(mergedConfig.Contexts) {
		nodeContextName := tview.NewTreeNode(" " + thisContextName)

		namespacesInThisContextsCluster, err := getNamespacesInContextsCluster(thisContextName)
		if err != nil {
			nodeContextName.SetColor(tcell.ColorRed).SetText(" " + thisContextName + " (" + err.Error() + ")").SetSelectable(false)
		} else if thisContextName == mergedConfig.CurrentContext {
			nodeContextName.SetColor(tcell.ColorGreen).SetText(" " + thisContextName + " (active)")
		} else {
			nodeContextName.SetColor(tcell.ColorTurquoise)
		}

		nodeContextName.Collapse()
		nodeContextName.SetSelectedFunc(func() {
			nodeContextName.SetExpanded(!nodeContextName.IsExpanded())

			if nodeContextName.IsExpanded() && expandedNode != nodeContextName {
				expandedNode.Collapse()
				expandedNode = nodeContextName
			}
		})

		nodeRoot.AddChild(nodeContextName)

		for _, thisNamespace := range namespacesInThisContextsCluster {
			nodeNamespace := tview.NewTreeNode(" " + thisNamespace.Name).SetReference(contextNamespaceTuple{thisContextName, thisNamespace.Name})

			if thisContextName == mergedConfig.CurrentContext {
				nodeContextName.Expand()
				expandedNode = nodeContextName

				if thisNamespace.Name == mergedConfig.Contexts[thisContextName].Namespace {
					nodeNamespace.SetColor(tcell.ColorGreen)
					highlightNode = nodeNamespace
				}
			}

			nodeNamespace.SetSelectedFunc(func() {
				app.Stop()
				switchContext(nodeNamespace.GetReference().(contextNamespaceTuple))
			})
			nodeContextName.AddChild(nodeNamespace)
		}

	}

	tree := tview.NewTreeView().SetRoot(nodeRoot).SetCurrentNode(highlightNode)

	if err := app.SetRoot(tree, true).Run(); err != nil {
		log.Fatalln(err)
	}
}

func getNamespacesInContextsCluster(k8sContext string) ([]corev1.Namespace, error) {

	config, err := clientcmd.NewDefaultClientConfig(*mergedConfig, &clientcmd.ConfigOverrides{CurrentContext: k8sContext}).ClientConfig()
	if err != nil {
		log.Fatalln(err)
	}

	config.Timeout = time.Second

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalln(err)
	}

	namespaces, err := clientset.CoreV1().Namespaces().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		switch err.(type) {
		case *url.Error:
			return []corev1.Namespace{}, fmt.Errorf("unreachable")
		case *apierrors.StatusError:
			return []corev1.Namespace{}, fmt.Errorf("error from api: " + err.Error())
		default:
			return []corev1.Namespace{}, fmt.Errorf("error")
		}
	}

	return namespaces.Items, nil
}

func switchContext(rh contextNamespaceTuple) {
	mergedConfig.CurrentContext = rh.k8sContext
	mergedConfig.Contexts[rh.k8sContext].Namespace = rh.k8sNamespace

	removeStaleContextConfigs()

	configAccess := clientcmd.NewDefaultClientConfigLoadingRules()

	if err := clientcmd.ModifyConfig(configAccess, *mergedConfig, false); err != nil {
		log.Fatalln(err)
	}

	log.Printf("switched to %s/%s", rh.k8sContext, rh.k8sNamespace)
}

func quickSwitch() {
	if len(os.Args) == 2 {
		if !namespaceExists(mergedConfig.CurrentContext, os.Args[1]) {
			log.Fatalf("namespace %s not found in context %s\n", os.Args[1], mergedConfig.CurrentContext)
		}

		switchContext(contextNamespaceTuple{mergedConfig.CurrentContext, os.Args[1]})
		os.Exit(0)
	}

	if len(os.Args) == 3 && os.Args[2] == "." {
		if !contextExists(os.Args[1]) || !namespaceExists(os.Args[1], "default") {
			log.Fatalf("namespace %s not found in context %s\n", "default", os.Args[1])
		}

		switchContext(contextNamespaceTuple{os.Args[1], "default"})
		os.Exit(0)
	}

	if len(os.Args) == 3 {
		if !contextExists(os.Args[1]) || !namespaceExists(os.Args[1], os.Args[2]) {
			log.Fatalf("namespace %s not found in context %s\n", os.Args[2], os.Args[1])
		}

		switchContext(contextNamespaceTuple{os.Args[1], os.Args[2]})
		os.Exit(0)
	}
}

func removeStaleContextConfigs() {

	for _, configFilename := range strings.Split(kubeconfLocation, ":") {
		var output []string

		cfStat, err := os.Stat(configFilename)
		if err != nil {
			log.Fatalln("could not stat kubeconfig files")
		}

		cfFileMode := cfStat.Mode()

		cfContent, err := ioutil.ReadFile(configFilename)
		if err != nil {
			log.Fatalln("could not read kubeconfig files")
		}
		cfLines := strings.Split(string(cfContent), "\n")

		for _, line := range cfLines {
			if strings.Contains(line, "current-context:") {
				continue
			}

			output = append(output, line)
		}

		if err := ioutil.WriteFile(configFilename, []byte(strings.Join(output, "\n")), cfFileMode); err != nil {
			log.Fatalln("could not update kubeconfig files")
		}
	}
}

func contextExists(k8sContext string) bool {
	_, exists := mergedConfig.Contexts[k8sContext]
	return exists
}

func namespaceExists(k8sContext, k8sNamespace string) bool {
	namespacesInThisContextsCluster, err := getNamespacesInContextsCluster(k8sContext)
	if err != nil {
		log.Fatalln(err)
	}

	for _, ns := range namespacesInThisContextsCluster {
		if ns.Name == k8sNamespace {
			return true
		}
	}

	return false
}

func mapKeysToSortedArray(m map[string]*clientcmdapi.Context) []string {
	var s []string

	for k := range m {
		s = append(s, k)
	}

	sort.Strings(s)
	return s
}

func printUsage() {
	usageText := `usage:
	
./kubeswitch                          select context/namespace graphically
./kubeswitch <namespace>              switch to namespace in current context quickly
./kubeswitch <context> <namespace>    switch to namespace in context quickly
./kubeswitch <context>/<namespace>    switch to namespace in context quickly`

	fmt.Println(usageText)
	os.Exit(2)
}
