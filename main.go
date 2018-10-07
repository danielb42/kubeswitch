package main

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/gdamore/tcell"
	"github.com/rivo/tview"
	yaml "gopkg.in/yaml.v2"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

type config struct {
	CurrentContextName string `yaml:"current-context"`
	Contexts           []struct {
		Name    string
		Context context
	}
}

type context struct {
	Cluster   string
	Namespace string
}

var (
	kubeconfig     config
	currentContext context
)

func getNamespaces(context string) []string {
	config, _ := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{
			ExplicitPath: os.Getenv("KUBECONFIG")},
		&clientcmd.ConfigOverrides{
			CurrentContext: context}).
		ClientConfig()

	clientset, _ := kubernetes.NewForConfig(config)
	namespaces, _ := clientset.CoreV1().Namespaces().List(v1.ListOptions{})

	var slc []string
	for _, thisNamespace := range namespaces.Items {
		slc = append(slc, thisNamespace.Name)
	}

	return slc
}

func switchContext(ctx context) {
	// TODO: really switch context
	fmt.Printf("kubectl config set-context %v --namespace=%v &>/dev/null \n", ctx.Cluster, ctx.Namespace)
}

func getContextNames() []string {
	var slc []string
	for _, thisContext := range kubeconfig.Contexts {
		slc = append(slc, thisContext.Name)

		if thisContext.Name == kubeconfig.CurrentContextName {
			currentContext.Cluster = thisContext.Context.Cluster
			currentContext.Namespace = thisContext.Context.Namespace
		}
	}

	return slc
}

func main() {
	configContent, _ := ioutil.ReadFile(os.Getenv("KUBECONFIG"))
	yaml.Unmarshal(configContent, &kubeconfig)

	nodeRoot := tview.NewTreeNode(".")
	var highlightNode *tview.TreeNode

	for _, thisContextName := range getContextNames() {
		nodeCluster := tview.NewTreeNode(thisContextName).
			SetSelectable(false)

		namespacesHere := getNamespaces(thisContextName)

		if len(namespacesHere) > 0 {
			nodeCluster.SetColor(tcell.ColorTurquoise)
		} else {
			nodeCluster.SetColor(tcell.ColorRed).
				SetText(thisContextName + " (unreachable)")
		}

		nodeRoot.AddChild(nodeCluster)

		for _, thisNamespace := range namespacesHere {

			nodeNamespace := tview.NewTreeNode(thisNamespace).
				SetReference(context{thisContextName, thisNamespace})

			if thisContextName == currentContext.Cluster &&
				thisNamespace == currentContext.Namespace {
				nodeCluster.SetColor(tcell.ColorGreen)
				nodeNamespace.SetColor(tcell.ColorGreen)
				highlightNode = nodeNamespace
			}

			nodeCluster.AddChild(nodeNamespace)
		}
	}

	app := tview.NewApplication()
	tree := tview.NewTreeView().
		SetRoot(nodeRoot).
		SetCurrentNode(highlightNode).
		SetTopLevel(1).
		SetSelectedFunc(func(node *tview.TreeNode) {
			app.Stop()
			switchContext(node.GetReference().(context))
		})

	app.SetRoot(tree, true).Run()
}
