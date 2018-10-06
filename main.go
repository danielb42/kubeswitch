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
	Contexts []struct {
		Name string `json:"name"`
	} `json:"contexts"`
}

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
	fmt.Printf("kubectl config set-context %v --namespace=%v &>/dev/null \n", ctx.cluster, ctx.namespace)
}

func getContexts() []string {
	configContent, _ := ioutil.ReadFile(os.Getenv("KUBECONFIG"))
	var kubeconfig kubeconf
	yaml.Unmarshal(configContent, &kubeconfig)

	var slc []string
	for _, context := range kubeconfig.Contexts {
		slc = append(slc, context.Name)
	}

	return slc
}

func main() {
	nodeRoot := tview.NewTreeNode(".")

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
			nodeNS := tview.NewTreeNode(namespace).
				SetReference(context{cluster, namespace})

			nodeCluster.AddChild(nodeNS)
		}
	}

	app := tview.NewApplication()
	tree := tview.NewTreeView().
		SetRoot(nodeRoot).
		SetCurrentNode(nodeRoot).
		SetTopLevel(1).
		SetSelectedFunc(func(node *tview.TreeNode) {
			app.Stop()
			switchContext(node.GetReference().(context))
		})

	app.SetRoot(tree, true).Run()
}
