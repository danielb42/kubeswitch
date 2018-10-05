package main

import (
	"github.com/gdamore/tcell"
	"github.com/rivo/tview"
)

type context struct {
	cluster   string
	namespace string
}

func getClusters() []string {
	return []string{"cluster1", "cluster2", "cluster3"}
}

func getNamespaces(cluster string) []string {
	switch cluster {
	case "cluster1":
		return []string{"namespace1", "namespace2"}
	case "cluster2":
		return []string{"namespace1", "namespace2", "namespace3", "namespace4"}
	case "cluster3":
		return []string{"namespace1", "namespace2", "namespace3", "namespace4", "namespace5"}
	case "cluster4":
		return []string{"namespace1", "namespace2", "namespace3"}
	}
	return []string{}
}

func switchContext(ctx context) {
	println("switch to cluster "+ctx.cluster+", namespace", ctx.namespace)
}

func main() {
	root := tview.NewTreeNode(".")

	for _, cluster := range getClusters() {
		nodeCluster := tview.NewTreeNode(cluster).
			SetSelectable(false).
			SetColor(tcell.ColorTurquoise)

		root.AddChild(nodeCluster)

		for _, namespace := range getNamespaces(cluster) {
			nodeNS := tview.NewTreeNode(namespace).
				SetReference(context{cluster, namespace})

			nodeCluster.AddChild(nodeNS)
		}
	}

	app := tview.NewApplication()
	tree := tview.NewTreeView().
		SetRoot(root).
		SetCurrentNode(root).
		SetTopLevel(1).
		SetSelectedFunc(func(node *tview.TreeNode) {
			app.Stop()
			switchContext(node.GetReference().(context))
		})

	app.SetRoot(tree, true).Run()
}
