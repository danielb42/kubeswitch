package main

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/gdamore/tcell"
	"github.com/rivo/tview"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"

	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
	_ "k8s.io/client-go/plugin/pkg/client/auth/openstack"
)

type referenceHelper struct {
	context   string
	namespace string
}

var mergedConfig *clientcmdapi.Config

func getNamespacesInContextsCluster(context string) ([]corev1.Namespace, error) {
	config, err := clientcmd.NewDefaultClientConfig(*mergedConfig, &clientcmd.ConfigOverrides{CurrentContext: context}).ClientConfig()

	if err != nil {
		log.Fatalln(err)
	}

	config.Timeout = time.Second

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalln(err)
	}

	namespaces, err := clientset.CoreV1().Namespaces().List(metav1.ListOptions{})
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

	for _, thisContextName := range mapKeysToSortedArray(mergedConfig.Contexts) {
		nodeContextName := tview.NewTreeNode(" " + thisContextName)

		namespacesInThisContextsCluster, err := getNamespacesInContextsCluster(thisContextName)
		if err != nil {
			nodeContextName.SetColor(tcell.ColorRed).
				SetText(" " + thisContextName + " (" + err.Error() + ")").
				SetSelectable(false)
		} else if thisContextName == mergedConfig.CurrentContext {
			nodeContextName.SetColor(tcell.ColorGreen).
				SetText(" " + thisContextName + " (active)")
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
				SetReference(referenceHelper{thisContextName, thisNamespace.Name})

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

func mapKeysToSortedArray(m map[string]*clientcmdapi.Context) []string {
	var s []string

	for k := range m {
		s = append(s, k)
	}

	sort.Strings(s)
	return s
}
