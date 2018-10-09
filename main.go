package main

import (
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"reflect"
	"time"

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
	config, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{
			ExplicitPath: os.Getenv("KUBECONFIG")},
		&clientcmd.ConfigOverrides{
			CurrentContext: context}).
		ClientConfig()
	if err != nil {
		if reflect.TypeOf(err).String() != "clientcmd.errConfigurationInvalid" {
			log.Fatalln(err)
		}
		return []k8s.Namespace{}
	}

	config.Timeout = 500 * time.Millisecond

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalln(err)
	}

	namespaces, err := clientset.CoreV1().Namespaces().List(v1.ListOptions{})
	if err != nil {
		if _, urlError := err.(*url.Error); !urlError {
			log.Fatalln(err)
		}
	}

	return namespaces.Items
}

func switchContext(rh referenceHelper) {
	config, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{
			ExplicitPath: os.Getenv("KUBECONFIG")},
		&clientcmd.ConfigOverrides{}).
		RawConfig()
	if err != nil {
		log.Fatalln(err)
	}

	config.CurrentContext = rh.context
	config.Contexts[rh.context].Namespace = rh.namespace
	configAccess := clientcmd.NewDefaultClientConfigLoadingRules()
	if err := clientcmd.ModifyConfig(configAccess, config, false); err != nil {
		log.Fatalln(err)
	}

	log.Printf("switched to %s/%s", rh.context, rh.namespace)
}

func loadConfig() {
	configContent, err := ioutil.ReadFile(os.Getenv("KUBECONFIG"))
	if err != nil {
		log.Fatalln(err)
	}

	if err := yaml.Unmarshal(configContent, &kubeconfig); err != nil {
		log.Fatalln(err)
	}
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

	if err := app.SetRoot(tree, true).Run(); err != nil {
		log.Fatalln(err)
	}
}
