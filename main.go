package main

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/gdamore/tcell"
	"github.com/rivo/tview"
	yaml "gopkg.in/yaml.v2"
	k8s "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

type config struct {
	ActiveContext string `yaml:"current-context"`
	Contexts      []struct {
		Name       string `yaml:"name"`
		Attributes struct {
			ActiveNamespace string `yaml:"namespace"`
		} `yaml:"context"`
	}
}

type referenceHelper struct {
	context   string
	namespace string
}

var kubeconfig config

func getNamespacesInContextsCluster(context string) []k8s.Namespace {
	config, _ := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{
			ExplicitPath: os.Getenv("KUBECONFIG")},
		&clientcmd.ConfigOverrides{
			CurrentContext: context}).
		ClientConfig()

	clientset, _ := kubernetes.NewForConfig(config)
	namespaces, _ := clientset.CoreV1().Namespaces().List(v1.ListOptions{})

	return namespaces.Items
}

func switchContext(ts twostrings) {
	// TODO: really switch context
	fmt.Printf(
		"kubectl config set-context %v --namespace=%v &>/dev/null \n",
		ts.a,
		ts.b)
}

func loadConfig() {
	configContent, _ := ioutil.ReadFile(os.Getenv("KUBECONFIG"))
	yaml.Unmarshal(configContent, &kubeconfig)
}

func main() {
	loadConfig()

	nodeRoot := tview.NewTreeNode(".")
	highlightNode := nodeRoot

	for _, thisContext := range kubeconfig.Contexts {
		nodeContextName := tview.NewTreeNode(thisContext.Name).
			SetSelectable(false)

		namespacesInThisContextsCluster := getNamespacesInContextsCluster(thisContext.Name)

		if len(namespacesInThisContextsCluster) == 0 {
			nodeContextName.SetColor(tcell.ColorRed).
				SetText(thisContext.Name + " (unreachable)")
		} else if thisContext.Name == kubeconfig.ActiveContext {
			nodeContextName.SetColor(tcell.ColorGreen).
				SetText(thisContext.Name + " (active)")
		} else {
			nodeContextName.SetColor(tcell.ColorTurquoise)
		}

		nodeRoot.AddChild(nodeContextName)

		for _, thisNamespace := range namespacesInThisContextsCluster {

			nodeNamespace := tview.NewTreeNode(thisNamespace.Name).
				SetReference(referenceHelper{thisContext.Name, thisNamespace.Name})

			if thisContext.Name == kubeconfig.ActiveContext &&
				thisNamespace.Name == thisContext.Attributes.ActiveNamespace {
				nodeNamespace.SetColor(tcell.ColorGreen)
				highlightNode = nodeNamespace
			}

			nodeContextName.AddChild(nodeNamespace)
		}
	}

	app := tview.NewApplication()
	tree := tview.NewTreeView().
		SetRoot(nodeRoot).
		SetCurrentNode(highlightNode).
		SetTopLevel(1).
		SetSelectedFunc(func(node *tview.TreeNode) {
			app.Stop()
			switchContext(node.GetReference().(referenceHelper))
		})

	app.SetRoot(tree, true).Run()
}
