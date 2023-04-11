/*
Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	replicatev1 "github.com/jnytnai0613/resource-replicator/api/v1"
	"github.com/jnytnai0613/resource-replicator/pkg/constants"
	"github.com/jnytnai0613/resource-replicator/pkg/healthcheck"
	"github.com/jnytnai0613/resource-replicator/pkg/kubeconfig"
)

// ClusterDetectorReconciler reconciles a ClusterDetector object
type ClusterDetectorReconciler struct {
	client.Client
}

// // Create a Custom Resource ClusterDetector and register the remote cluster status.
func SetupClusterDetector(localClient client.Client, log logr.Logger) error {
	_, apiServer, targetCluster, err := kubeconfig.ReadKubeconfig(localClient)
	if err != nil {
		return err
	}

	ctx := context.Background()

	for _, target := range targetCluster {
		clusterDetector := &replicatev1.ClusterDetector{}
		clusterDetector.SetNamespace("resource-replicator-system")
		// .Metadata.Name must be a lowercase RFC 1123 subdomain must consist of lower case alphanumeric
		// characters, '-' or '.', and must start and end with an alphanumeric character (e.g. 'example.com',
		// regex used for validation is [a-z0-9]([-a-z0-9]*[a-z0-9])?(\\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*').
		//
		// If ContextName is put in .Metadata.Name, it will be trapped by the above restriction,
		// so the format is "ClusterName.UserName".
		clusterDetector.SetName(fmt.Sprintf("%s.%s", target.ClusterName, target.UserName))

		/////////////////////////////
		// Create ClusterDetector
		/////////////////////////////
		var (
			apiEndpoint string
			endpoint    corev1.Endpoints
			role        = make(map[string]string)
		)

		// Determine the role of the cluster.
		// The local cluster is the primary and the others are the workers.
		// The local cluster is determined by the address and port obtained
		// from the kubernetes endpoint in the default namespace.
		if err := localClient.Get(ctx,
			client.ObjectKey{Namespace: "default", Name: "kubernetes"}, &endpoint); err != nil {
			return err
		}
		for _, subsets := range endpoint.Subsets {
			var (
				apiAddress string
				apiPort    string
			)
			for _, a := range subsets.Addresses {
				apiAddress = a.IP
			}

			for _, p := range subsets.Ports {
				apiPort = fmt.Sprint(p.Port)
			}

			apiEndpoint = fmt.Sprintf("https://%s:%s", apiAddress, apiPort)
			break
		}
		for _, a := range apiServer {
			if target.ClusterName == a.Name {
				if apiEndpoint == a.Endpoint {
					role["app.kubernetes.io/role"] = "primary"
				} else {
					role["app.kubernetes.io/role"] = "secondary"
				}
				break
			}
		}

		if op, err := ctrl.CreateOrUpdate(ctx, localClient, clusterDetector, func() error {
			clusterDetector.Labels = role
			clusterDetector.Spec.Context = target.ContextName
			clusterDetector.Spec.Cluster = target.ClusterName
			clusterDetector.Spec.User = target.UserName
			return nil
		}); op != controllerutil.OperationResultNone {
			log.Info(fmt.Sprintf("[ClusterDetector: %s] %s", clusterDetector.GetName(), op))
		} else if err != nil {
			return err
		}

		/////////////////////////////
		// Update Status
		/////////////////////////////
		var (
			currentClusterStatus string
			nextClusterStatus    string
		)
		if err := localClient.Get(ctx,
			client.ObjectKey{Namespace: constants.Namespace, Name: clusterDetector.GetName()},
			clusterDetector); err != nil {
			// If the resource does not exist, create it.
			// Therefore, Not Found errors are ignored.
			if !errors.IsNotFound(err) {
				return err
			}
		}
		currentClusterStatus = clusterDetector.Status.ClusterStatus

		for _, a := range apiServer {
			if target.ClusterName == a.Name {
				// Verify that you can communicate with the remote Kubernetes cluster.
				if err := healthcheck.HealthChecks(a); err != nil {
					clusterDetector.Status.Reason = fmt.Sprintf("%s", err)
					if currentClusterStatus != "Unknown" {
						log.Error(err, fmt.Sprintf("[Cluster: %s] Health Check failed.", a.Name))
					}
					nextClusterStatus = "Unknown"
					break
				}
				clusterDetector.Status.Reason = ""
				nextClusterStatus = "Running"
				break
			}
		}
		clusterDetector.Status.ClusterStatus = nextClusterStatus
		if err := localClient.Status().Update(ctx, clusterDetector); err != nil {
			return err
		}

		if currentClusterStatus != nextClusterStatus {
			log.Info(fmt.Sprintf("[ClusterDetector: %s] Status update completed.", clusterDetector.GetName()))
		}
	}

	return nil
}

//+kubebuilder:rbac:groups=replicate.jnytnai0613.github.io,resources=clusterdetectors,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=replicate.jnytnai0613.github.io,resources=clusterdetectors/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=replicate.jnytnai0613.github.io,resources=clusterdetectors/finalizers,verbs=update
//+kubebuilder:rbac:groups=core,resources=endpoints,verbs=get
//+kubebuilder:rbac:groups=core,resources=secrets,verbs=get

// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.6/pkg/reconcile
func (r *ClusterDetectorReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	if err := SetupClusterDetector(r.Client, logger); err != nil {
		logger.Error(err, "Failed to initialize ClusterDetector.")
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ClusterDetectorReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&replicatev1.ClusterDetector{}).
		Complete(r)
}
