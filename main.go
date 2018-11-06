package main

import (
	"errors"
	"fmt"
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
	apierrors "k8s.io/apimachinery/pkg/api/errors"
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
	} `yaml:"contexts"`
}

type referenceHelper struct {
	context   string
	namespace string
}

var kubeconfig config

func getNamespacesInContextsCluster(context string) ([]k8s.Namespace, error) {
	config, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{
			ExplicitPath: os.Getenv("KUBECONFIG")},
		&clientcmd.ConfigOverrides{
			CurrentContext: context}).
		ClientConfig()

	if err != nil {
		if reflect.TypeOf(err).String() == "clientcmd.errConfigurationInvalid" {
			return []k8s.Namespace{}, fmt.Errorf("error in config file")
		}

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

	if len(configContent) == 0 {
		log.Fatalln(errors.New("empty configuration file"))
	}

	if err := yaml.Unmarshal(configContent, &kubeconfig); err != nil {
		log.Fatalln(err)
	}
}

func main() {
	loadConfig()

	app := tview.NewApplication()

	nodeRoot := tview.NewTreeNode("Contexts").
		SetSelectable(false)

	expandedNode := new(tview.TreeNode)
	highlightNode := nodeRoot

	for _, thisContext := range kubeconfig.Contexts {
		nodeContextName := tview.NewTreeNode(" " + thisContext.Name)

		namespacesInThisContextsCluster, err := getNamespacesInContextsCluster(thisContext.Name)
		if err != nil {
			nodeContextName.SetColor(tcell.ColorRed).
				SetText(" " + thisContext.Name + " (" + err.Error() + ")").
				SetSelectable(false)
		} else if thisContext.Name == kubeconfig.ActiveContext {
			nodeContextName.SetColor(tcell.ColorGreen).
				SetText(" " + thisContext.Name + " (active)")
		} else {
			nodeContextName.SetColor(tcell.ColorTurquoise)
		}

		nodeContextName.Collapse()
		nodeContextName.SetSelectedFunc(func() {
			nodeContextName.SetExpanded(!nodeContextName.IsExpanded())

			if nodeContextName.IsExpanded() && expandedNode != nodeContextName {
				expandedNode.SetExpanded(false)
				expandedNode = nodeContextName
			}
		})

		nodeRoot.AddChild(nodeContextName)

		for _, thisNamespace := range namespacesInThisContextsCluster {
			nodeNamespace := tview.NewTreeNode(" " + thisNamespace.Name).
				SetReference(referenceHelper{thisContext.Name, thisNamespace.Name})

			if thisContext.Name == kubeconfig.ActiveContext {
				nodeContextName.SetExpanded(true)
				expandedNode = nodeContextName

				if thisNamespace.Name == thisContext.Attributes.ActiveNamespace {
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
