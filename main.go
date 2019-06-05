package main

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/gdamore/tcell"
	"github.com/rivo/tview"

	k8s "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

type referenceHelper struct {
	context   string
	namespace string
}

var mergedConfig *clientcmdapi.Config

func getNamespacesInContextsCluster(context string) ([]k8s.Namespace, error) {
	config, err := clientcmd.NewDefaultClientConfig(*mergedConfig, &clientcmd.ConfigOverrides{CurrentContext: context}).ClientConfig()

	if err != nil {
		log.Fatalln(err)
	}

	config.Timeout = 500 * time.Millisecond

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalln(err)
	}

	namespaces, err := clientset.CoreV1().Namespaces().List(v1.ListOptions{})
	if err != nil {
		switch err.(type) {
		case *url.Error:
			return []k8s.Namespace{}, fmt.Errorf("unreachable")
		case *apierrors.StatusError:
			return []k8s.Namespace{}, fmt.Errorf("error from api: " + err.(*apierrors.StatusError).Error())
		default:
			return []k8s.Namespace{}, fmt.Errorf("error")
		}
	}

	return namespaces.Items, nil
}

func switchContext(rh referenceHelper) {
	mergedConfig.CurrentContext = rh.context
	mergedConfig.Contexts[rh.context].Namespace = rh.namespace

	configAccess := clientcmd.NewDefaultClientConfigLoadingRules()
	if err := clientcmd.ModifyConfig(configAccess, *mergedConfig, false); err != nil {
		log.Fatalln(err)
	}

	log.Printf("switched to %s/%s", rh.context, rh.namespace)
}

func quickSwitch() {
	if len(os.Args) == 1 {
		return
	}

	s := strings.Split(os.Args[1], "/")

	if len(os.Args) == 2 && len(s) == 1 && namespaceExists(mergedConfig.CurrentContext, os.Args[1]) {
		switchContext(referenceHelper{mergedConfig.CurrentContext, os.Args[1]})
		os.Exit(0)
	}

	if len(os.Args) == 2 && len(s) == 2 && contextExists(s[0]) && namespaceExists(s[0], s[1]) {
		switchContext(referenceHelper{s[0], s[1]})
		os.Exit(0)
	}

	if len(os.Args) == 3 && contextExists(os.Args[1]) && namespaceExists(os.Args[1], os.Args[2]) {
		switchContext(referenceHelper{os.Args[1], os.Args[2]})
		os.Exit(0)
	}
}

func contextExists(context string) bool {
	_, exists := mergedConfig.Contexts[context]
	return exists
}

func namespaceExists(context, namespace string) bool {
	namespacesInThisContextsCluster, _ := getNamespacesInContextsCluster(context)

	for _, ns := range namespacesInThisContextsCluster {
		if ns.Name == namespace {
			return true
		}
	}

	return false
}

func main() {
	var err error

	kubeconfLocation := os.Getenv("HOME") + "/.kube/config"

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

	nodeRoot := tview.NewTreeNode("Contexts").
		SetSelectable(false)

	expandedNode := new(tview.TreeNode)
	highlightNode := nodeRoot

	for thisContext := range mergedConfig.Contexts {
		nodeContextName := tview.NewTreeNode(" " + thisContext)

		namespacesInThisContextsCluster, err := getNamespacesInContextsCluster(thisContext)
		if err != nil {
			nodeContextName.SetColor(tcell.ColorRed).
				SetText(" " + thisContext + " (" + err.Error() + ")").
				SetSelectable(false)
		} else if thisContext == mergedConfig.CurrentContext {
			nodeContextName.SetColor(tcell.ColorGreen).
				SetText(" " + thisContext + " (active)")
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
			nodeNamespace := tview.NewTreeNode(" " + thisNamespace.Name).
				SetReference(referenceHelper{thisContext, thisNamespace.Name})

			if thisContext == mergedConfig.CurrentContext {
				nodeContextName.Expand()
				expandedNode = nodeContextName

				if thisNamespace.Name == mergedConfig.Contexts[thisContext].Namespace {
					nodeNamespace.SetColor(tcell.ColorGreen)
					highlightNode = nodeNamespace
				}
			}

			nodeNamespace.SetSelectedFunc(func() {
				app.Stop()
				switchContext(nodeNamespace.GetReference().(referenceHelper))
			})
			nodeContextName.AddChild(nodeNamespace)
		}

	}

	tree := tview.NewTreeView().
		SetRoot(nodeRoot).
		SetCurrentNode(highlightNode)

	if err := app.SetRoot(tree, true).Run(); err != nil {
		log.Fatalln(err)
	}
}
