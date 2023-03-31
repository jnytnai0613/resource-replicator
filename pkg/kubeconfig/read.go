package kubeconfig

import (
	"bufio"
	"bytes"
	"context"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Used to ping a remote Kubernetes cluster to verify that the remote Kubernetes cluster
// is operating properly by sending a request to the endpoint.
type RemoteAPIServer struct {
	Name     string
	Endpoint string
}

// It is used for the following purposes
//   - Generate a client set using ContextName
//   - Set fields in Custom Resource ClusterDetector and register information
//     for all remote Kubernetes clusters in the ClusterDetector.
type TargetCluster struct {
	ContextName string
	ClusterName string
	UserName    string
}

// Read the kubeconfig file registered for the Secret Resource and register the necessary information
// for apiServers and clusters, respectively. At this time, a slice element is created for
// each Kubernetes cluster. kubeconfig must be created in advance as a secret resource.
func ReadKubeconfig(cli client.Client) (
	string,
	[]RemoteAPIServer,
	[]TargetCluster,
	error,
) {
	var secret corev1.Secret
	if err := cli.Get(
		context.Background(),
		client.ObjectKey{Namespace: "kubeconfig", Name: "config"},
		&secret,
	); err != nil {
		return "", nil, nil, err
	}

	m := secret.Data
	kubeconfig := string(m["config"])
	buf := bytes.NewBufferString(kubeconfig)
	scanner := bufio.NewScanner(buf)

	var (
		clusterName []string
		contextName []string
		endPoint    []string
		apiServers  []RemoteAPIServer
		clusters    []TargetCluster
		serverName  []string
		userName    []string
	)
scan:
	for scanner.Scan() {
		// Obtain information to register with RemoteAPIServer
		switch {
		case strings.Contains(scanner.Text(), "server:"):
			endPoint = strings.Fields(scanner.Text())
		case strings.Contains(scanner.Text(), "name:"):
			serverName = strings.Fields(scanner.Text())

			a := RemoteAPIServer{
				Name:     serverName[1],
				Endpoint: endPoint[1],
			}
			apiServers = append(apiServers, a)
		}

		// "context:", when this string is detected, start collecting context
		// (remote Kubernetes cluster) information.
		if !strings.Contains(scanner.Text(), "contexts:") {
			continue
		}

		// Obtain information to register with TargetCluster
		for scanner.Scan() {
			switch {
			// If the string "current-context" is detected, finish reading the kubeconfig file.
			case strings.Contains(scanner.Text(), "current-context:"):
				break scan
			case strings.Contains(scanner.Text(), "cluster:"):
				clusterName = strings.Fields(scanner.Text())
			case strings.Contains(scanner.Text(), "user:"):
				userName = strings.Fields(scanner.Text())
			case strings.Contains(scanner.Text(), "name:"):
				contextName = strings.Fields(scanner.Text())

				c := TargetCluster{
					ContextName: contextName[1],
					ClusterName: clusterName[1],
					UserName:    userName[1],
				}
				clusters = append(clusters, c)
			}
		}
	}

	return kubeconfig, apiServers, clusters, nil
}
