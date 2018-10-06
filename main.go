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

type context struct {
	cluster   string
	namespace string
}

type kubeconf struct {
	CurrentContextName      string `json:"current-context"`
	currentContextCluster   string
	currentContextNamespace string
	Contexts                []struct {
		Name      string `json:"name"`
		Cluster   string `json:"context.cluster"`
		Namespace string `json:"context.namespace"`
	} `json:"contexts"`
}

var kubeconfig kubeconf

func getNamespaces(context string) []string {
	config, _ := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: os.Getenv("KUBECONFIG")},
		&clientcmd.ConfigOverrides{
			CurrentContext: context,
		}).ClientConfig()

	clientset, _ := kubernetes.NewForConfig(config)
	namespaces, _ := clientset.CoreV1().Namespaces().List(v1.ListOptions{})

	var slc []string
	for _, namespace := range namespaces.Items {
		slc = append(slc, namespace.Name)
	}

	return slc
}

func switchContext(ctx context) {
	// TODO: really switch context
	fmt.Printf("kubectl config set-context %v --namespace=%v &>/dev/null \n", ctx.cluster, ctx.namespace)
}

func getContexts() []string {
	var slc []string
	for _, context := range kubeconfig.Contexts {
		slc = append(slc, context.Name)

		if context.Name == kubeconfig.CurrentContextName {
			kubeconfig.currentContextCluster = context.Cluster
			kubeconfig.currentContextNamespace = context.Namespace
		}
	}

	return slc
}

func main() {
	configContent, _ := ioutil.ReadFile(os.Getenv("KUBECONFIG"))
	yaml.Unmarshal(configContent, &kubeconfig)

	nodeRoot := tview.NewTreeNode(".")
	var highlightNode *tview.TreeNode

	for _, cluster := range getContexts() {
		nodeCluster := tview.NewTreeNode(cluster).
			SetSelectable(false)

		namespacesHere := getNamespaces(cluster)

		if len(namespacesHere) > 0 {
			nodeCluster.SetColor(tcell.ColorTurquoise)
		} else {
			nodeCluster.SetColor(tcell.ColorRed).
				SetText(cluster + " (unreachable)")
		}

		nodeRoot.AddChild(nodeCluster)

		for _, namespace := range namespacesHere {

			nodeNS := tview.NewTreeNode(namespace).SetReference(context{cluster, namespace})

			if cluster == kubeconfig.currentContextCluster &&
				namespace == kubeconfig.currentContextNamespace {
				nodeNS.SetColor(tcell.ColorGreen)
				highlightNode = nodeNS
			}

			nodeCluster.AddChild(nodeNS)
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
