package client

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/go-logr/logr"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/jnytnai0613/resource-replicator/pkg/kubeconfig"
)

// Used to create and update local Custom Resource.
func CreateLocalClient(log logr.Logger, scheme runtime.Scheme) (client.Client, error) {
	clientConfig := ctrl.GetConfigOrDie()
	kubeClient, err := client.New(clientConfig, client.Options{Scheme: &scheme})
	if err != nil {
		return nil, err
	}

	return kubeClient, nil
}

// Used to create, delete, and update Resource on the Kubernetes cluster to which it is Replicationed.
// Suck up the contexts from the kubeconfig file that aggregates the config of the target cluster,
// and create a clientset for each context. kubeconfig must be created in advance as a secret resource.
func CreateClientSets(cli client.Client) (map[string]*kubernetes.Clientset, error) {
	// The kubeconfig.ReadKubeconfig function retrieves the kubeconfig file that aggregates
	// the replication destination config and the context information read.
	config, _, clusters, err := kubeconfig.ReadKubeconfig(cli)
	if err != nil {
		return nil, err
	}

	configPath := filepath.Join(os.TempDir(), "config")
	f, err := os.Create(configPath)
	if err != nil {
		return nil, err
	}
	_, err = f.WriteString(config)
	if err != nil {
		return nil, err
	}

	// Specify the path of the kubeconfig file to be loaded in clientcmd.ClientConfigLoadingRules.
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	loadingRules.ExplicitPath = configPath

	// Generate as many client sets as the number of contexts (remote Kubernetes clusters) read from kubeconfig.
	clientsets := make(map[string]*kubernetes.Clientset)
	for _, t := range clusters {

		overrides := clientcmd.ConfigOverrides{
			CurrentContext: t.ContextName,
		}
		config := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, &overrides)
		clientConfig, err := config.ClientConfig()
		if err != nil {
			return nil, err
		}

		cs, err := kubernetes.NewForConfig(clientConfig)
		if err != nil {
			return nil, err
		}
		clientsets[fmt.Sprintf("%s.%s", t.ClusterName, t.UserName)] = cs
	}

	return clientsets, nil
}
