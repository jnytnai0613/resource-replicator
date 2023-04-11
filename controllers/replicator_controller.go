/*
MIT License
Copyright (c) 2023 Junya Taniai

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
*/

package controllers

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"go.uber.org/multierr"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	appsv1apply "k8s.io/client-go/applyconfigurations/apps/v1"
	corev1apply "k8s.io/client-go/applyconfigurations/core/v1"
	metav1apply "k8s.io/client-go/applyconfigurations/meta/v1"
	networkv1apply "k8s.io/client-go/applyconfigurations/networking/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	replicatev1 "github.com/jnytnai0613/resource-replicator/api/v1"
	"github.com/jnytnai0613/resource-replicator/pkg/constants"
	"github.com/jnytnai0613/resource-replicator/pkg/pki"
)

// ReplicatorReconciler reconciles a Replicator object
type ReplicatorReconciler struct {
	client.Client
	ClientSets map[string]*kubernetes.Clientset
	Recorder   record.EventRecorder
	Scheme     *runtime.Scheme
}

type ReplicateRuntime struct {
	ClientSet  *kubernetes.Clientset
	IsPrimary  bool
	Context    context.Context
	Log        logr.Logger
	Cluster    string
	Replicator replicatev1.Replicator
	Request    reconcile.Request
}

var (
	syncStatus []replicatev1.PerResourceApplyStatus
	replicator replicatev1.Replicator
)

// Create OwnerReference with CR as Owner
func createOwnerReferences(
	log logr.Logger,
	scheme *runtime.Scheme,
) (*metav1apply.OwnerReferenceApplyConfiguration,
	error,
) {
	gvk, err := apiutil.GVKForObject(&replicator, scheme)
	if err != nil {
		log.Error(err, "Unable get GVK")
		return nil, err
	}

	owner := metav1apply.OwnerReference().
		WithAPIVersion(gvk.GroupVersion().String()).
		WithKind(gvk.Kind).
		WithName(replicator.GetName()).
		WithUID(replicator.GetUID()).
		WithBlockOwnerDeletion(true).
		WithController(true)

	return owner, nil
}

func (r *ReplicatorReconciler) applyConfigMap(
	applyRuntime ReplicateRuntime,
	fieldMgr string,
) error {
	var (
		configMapClient = applyRuntime.ClientSet.CoreV1().ConfigMaps(applyRuntime.Replicator.Spec.ReplicationNamespace)
		log             = applyRuntime.Log
	)

	nextConfigMapApplyConfig := corev1apply.ConfigMap(
		applyRuntime.Replicator.Spec.ConfigMapName,
		applyRuntime.Replicator.Spec.ReplicationNamespace).
		WithData(applyRuntime.Replicator.Spec.ConfigMapData)

	if applyRuntime.IsPrimary {
		owner, err := createOwnerReferences(log, r.Scheme)
		if err != nil {
			log.Error(err, "Unable create OwnerReference")
			return err
		}
		nextConfigMapApplyConfig.WithOwnerReferences(owner)
	}

	configMap, err := configMapClient.Get(
		applyRuntime.Context,
		applyRuntime.Replicator.Spec.ConfigMapName,
		metav1.GetOptions{},
	)
	if err != nil {
		// If the resource does not exist, create it.
		// Therefore, Not Found errors are ignored.
		if !errors.IsNotFound(err) {
			return err
		}
	}
	currConfigMapApplyConfig, err := corev1apply.ExtractConfigMap(configMap, fieldMgr)
	if err != nil {
		return err
	}

	kind := *nextConfigMapApplyConfig.Kind
	name := *nextConfigMapApplyConfig.Name
	applyStatus := "applied"
	s := replicatev1.PerResourceApplyStatus{
		Cluster:     applyRuntime.Cluster,
		Kind:        kind,
		Name:        name,
		ApplyStatus: applyStatus,
	}
	if equality.Semantic.DeepEqual(currConfigMapApplyConfig, nextConfigMapApplyConfig) {
		syncStatus = append(syncStatus, s)
		return nil
	}

	applied, err := configMapClient.Apply(
		applyRuntime.Context,
		nextConfigMapApplyConfig,
		metav1.ApplyOptions{
			FieldManager: fieldMgr,
			Force:        true,
		},
	)
	if err != nil {
		applyStatus = "not applied"
		s.ApplyStatus = applyStatus
		syncStatus = append(syncStatus, s)

		log.Error(err, "unable to apply")
		return err
	}

	syncStatus = append(syncStatus, s)

	log.Info(fmt.Sprintf("Nginx ConfigMap Applied: [cluster] %s, [resource] %s", applyRuntime.Cluster, applied.GetName()))

	return nil
}

func (r *ReplicatorReconciler) applyDeployment(
	applyRuntime ReplicateRuntime,
	fieldMgr string,
) error {
	var (
		deploymentClient = applyRuntime.ClientSet.AppsV1().Deployments(applyRuntime.Replicator.Spec.ReplicationNamespace)
		labels           = map[string]string{"apps": "nginx"}
		log              = applyRuntime.Log
	)

	nextDeploymentApplyConfig := appsv1apply.Deployment(
		applyRuntime.Replicator.Spec.DeploymentName,
		applyRuntime.Replicator.Spec.ReplicationNamespace).
		WithSpec(appsv1apply.DeploymentSpec().
			WithSelector(metav1apply.LabelSelector().
				WithMatchLabels(labels)))

	if applyRuntime.Replicator.Spec.DeploymentSpec.Replicas != nil {
		replicas := *applyRuntime.Replicator.Spec.DeploymentSpec.Replicas
		nextDeploymentApplyConfig.Spec.WithReplicas(replicas)
	}

	if applyRuntime.Replicator.Spec.DeploymentSpec.Strategy != nil {
		types := *applyRuntime.Replicator.Spec.DeploymentSpec.Strategy.Type
		rollingUpdate := applyRuntime.Replicator.Spec.DeploymentSpec.Strategy.RollingUpdate
		nextDeploymentApplyConfig.Spec.WithStrategy(appsv1apply.DeploymentStrategy().
			WithType(types).
			WithRollingUpdate(rollingUpdate))
	}

	podTemplate := applyRuntime.Replicator.Spec.DeploymentSpec.Template
	podTemplate.WithLabels(labels)

	nextDeploymentApplyConfig.Spec.WithTemplate(podTemplate)

	if applyRuntime.IsPrimary {
		owner, err := createOwnerReferences(log, r.Scheme)
		if err != nil {
			log.Error(err, "Unable create OwnerReference")
			return err
		}
		nextDeploymentApplyConfig.WithOwnerReferences(owner)
	}

	// If EmptyDir is not set to Medium or SizeLimit, applyconfiguration
	// returns an empty pointer address. Therefore, a comparison between
	// applyconfiguration in subsequent steps will always detect a difference.
	// In the following process, if neither Medium nor SizeLimit is set,
	// explicitly set nil to prevent the above problem.
	for i, v := range nextDeploymentApplyConfig.Spec.Template.Spec.Volumes {
		e := v.EmptyDir
		if e != nil {
			if v.EmptyDir.Medium != nil || v.EmptyDir.SizeLimit != nil {
				break
			}
			nextDeploymentApplyConfig.Spec.Template.Spec.Volumes[i].
				WithEmptyDir(nil)
		}
	}

	deployment, err := deploymentClient.Get(
		applyRuntime.Context,
		applyRuntime.Replicator.Spec.DeploymentName,
		metav1.GetOptions{},
	)
	if err != nil {
		// If the resource does not exist, create it.
		// Therefore, Not Found errors are ignored.
		if !errors.IsNotFound(err) {
			return err
		}
	}
	currDeploymentMapApplyConfig, err := appsv1apply.ExtractDeployment(deployment, fieldMgr)
	if err != nil {
		return err
	}

	kind := *nextDeploymentApplyConfig.Kind
	name := *nextDeploymentApplyConfig.Name
	applyStatus := "applied"
	s := replicatev1.PerResourceApplyStatus{
		Cluster:     applyRuntime.Cluster,
		Kind:        kind,
		Name:        name,
		ApplyStatus: applyStatus,
	}
	if equality.Semantic.DeepEqual(currDeploymentMapApplyConfig, nextDeploymentApplyConfig) {
		syncStatus = append(syncStatus, s)
		return nil
	}

	applied, err := deploymentClient.Apply(
		applyRuntime.Context,
		nextDeploymentApplyConfig,
		metav1.ApplyOptions{
			FieldManager: fieldMgr,
			Force:        true,
		},
	)
	if err != nil {
		applyStatus = "not applied"
		s.ApplyStatus = applyStatus
		syncStatus = append(syncStatus, s)

		log.Error(err, "unable to apply")
		return err
	}

	syncStatus = append(syncStatus, s)

	log.Info(fmt.Sprintf("Nginx Deployment Applied: [cluster] %s, [resource] %s", applyRuntime.Cluster, applied.GetName()))

	return nil
}

func (r *ReplicatorReconciler) applyService(
	applyRuntime ReplicateRuntime,
	fieldMgr string,
) error {
	var (
		serviceClient = applyRuntime.ClientSet.CoreV1().Services(applyRuntime.Replicator.Spec.ReplicationNamespace)
		labels        = map[string]string{"apps": "nginx"}
		log           = applyRuntime.Log
	)

	nextServiceApplyConfig := corev1apply.Service(
		applyRuntime.Replicator.Spec.ServiceName,
		applyRuntime.Replicator.Spec.ReplicationNamespace).
		WithSpec((*corev1apply.ServiceSpecApplyConfiguration)(applyRuntime.Replicator.Spec.ServiceSpec).
			WithSelector(labels))

	if applyRuntime.IsPrimary {
		owner, err := createOwnerReferences(log, r.Scheme)
		if err != nil {
			log.Error(err, "Unable create OwnerReference")
			return err
		}
		nextServiceApplyConfig.WithOwnerReferences(owner)
	}

	service, err := serviceClient.Get(
		applyRuntime.Context,
		applyRuntime.Replicator.Spec.ServiceName,
		metav1.GetOptions{},
	)
	if err != nil {
		// If the resource does not exist, create it.
		// Therefore, Not Found errors are ignored.
		if !errors.IsNotFound(err) {
			return err
		}
	}
	currServiceApplyConfig, err := corev1apply.ExtractService(service, fieldMgr)
	if err != nil {
		return err
	}

	kind := *nextServiceApplyConfig.Kind
	name := *nextServiceApplyConfig.Name
	applyStatus := "applied"
	s := replicatev1.PerResourceApplyStatus{
		Cluster:     applyRuntime.Cluster,
		Kind:        kind,
		Name:        name,
		ApplyStatus: applyStatus,
	}
	if equality.Semantic.DeepEqual(currServiceApplyConfig, nextServiceApplyConfig) {
		syncStatus = append(syncStatus, s)
		return nil
	}

	applied, err := serviceClient.Apply(
		applyRuntime.Context,
		nextServiceApplyConfig,
		metav1.ApplyOptions{
			FieldManager: fieldMgr,
			Force:        true,
		},
	)
	if err != nil {
		applyStatus = "not applied"
		s.ApplyStatus = applyStatus
		syncStatus = append(syncStatus, s)

		log.Error(err, "unable to apply")
		return err
	}

	syncStatus = append(syncStatus, s)

	log.Info(fmt.Sprintf("Nginx Service Applied: [cluster] %s, [resource] %s", applyRuntime.Cluster, applied.GetName()))

	return nil
}

func (r *ReplicatorReconciler) applyIngress(
	applyRuntime ReplicateRuntime,
	fieldMgr string,
) error {
	var (
		annotateRewriteTarget = map[string]string{"nginx.ingress.kubernetes.io/rewrite-target": "/"}
		annotateVerifyClient  = map[string]string{"nginx.ingress.kubernetes.io/auth-tls-verify-client": "on"}
		annotateTlsSecret     = map[string]string{"nginx.ingress.kubernetes.io/auth-tls-secret": fmt.Sprintf("%s/%s", applyRuntime.Replicator.Spec.ReplicationNamespace, constants.IngressSecretName)}
		ingressClient         = applyRuntime.ClientSet.NetworkingV1().Ingresses(applyRuntime.Replicator.Spec.ReplicationNamespace)
		log                   = applyRuntime.Log
		secretClient          = applyRuntime.ClientSet.CoreV1().Secrets(applyRuntime.Replicator.Spec.ReplicationNamespace)
	)

	nextIngressApplyConfig := networkv1apply.Ingress(
		applyRuntime.Replicator.Spec.IngressName,
		applyRuntime.Replicator.Spec.ReplicationNamespace).
		WithAnnotations(annotateRewriteTarget).
		WithSpec((*networkv1apply.IngressSpecApplyConfiguration)(applyRuntime.Replicator.Spec.IngressSpec).
			WithIngressClassName(constants.IngressClassName))

	ingress, err := ingressClient.Get(
		applyRuntime.Context,
		applyRuntime.Replicator.Spec.IngressName,
		metav1.GetOptions{},
	)
	if err != nil {
		// If the resource does not exist, create it.
		// Therefore, Not Found errors are ignored.
		if !errors.IsNotFound(err) {
			return err
		}
	}

	if applyRuntime.Replicator.Spec.IngressSecureEnabled {
		// Re-create Secret if 'spec.tls[].hosts[]' has changed
		if len(ingress.Spec.TLS) > 0 {
			secrets, err := secretClient.List(
				applyRuntime.Context,
				metav1.ListOptions{},
			)
			if err != nil {
				return err
			}

			ih := ingress.Spec.TLS[0].Hosts[0]
			sh := *applyRuntime.Replicator.Spec.IngressSpec.Rules[0].Host
			if ih != sh {
				log.Info("Host is not different")
				for _, secret := range secrets.Items {
					if err := secretClient.Delete(
						applyRuntime.Context,
						secret.GetName(),
						metav1.DeleteOptions{},
					); err != nil {
						return err
					}
					log.Info(fmt.Sprintf("delete Secret resource: %s", secret.GetName()))
				}
			}
		}

		if err := r.applyIngressSecret(
			applyRuntime,
			constants.FieldManager,
		); err != nil {
			log.Error(err, "Unable create Ingress Secret")
			return err
		}

		if err := r.applyClientSecret(
			applyRuntime,
			constants.FieldManager,
		); err != nil {
			log.Error(err, "Unable create Client Secret")
			return err
		}

		nextIngressApplyConfig.
			WithAnnotations(annotateVerifyClient).
			WithAnnotations(annotateTlsSecret).
			Spec.
			WithTLS(networkv1apply.IngressTLS().
				WithHosts(*applyRuntime.Replicator.Spec.IngressSpec.Rules[0].Host).
				WithSecretName(constants.IngressSecretName))
	}

	// TODO:
	// Need to consider duplicate checks.
	if len(nextIngressApplyConfig.Spec.TLS) > 0 {
		// When replicating to multiple targets, the TLS field may contain multiple identical values.
		// Therefore, only one value is allowed.
		nextIngressApplyConfig.Spec.TLS = nextIngressApplyConfig.Spec.TLS[:1]
	}

	if applyRuntime.IsPrimary {
		owner, err := createOwnerReferences(log, r.Scheme)
		if err != nil {
			log.Error(err, "Unable create OwnerReference")
			return err
		}
		nextIngressApplyConfig.WithOwnerReferences(owner)
	}

	currIngressApplyConfig, err := networkv1apply.ExtractIngress(ingress, fieldMgr)
	if err != nil {
		return err
	}

	kind := *nextIngressApplyConfig.Kind
	name := *nextIngressApplyConfig.Name
	applyStatus := "applied"
	s := replicatev1.PerResourceApplyStatus{
		Cluster:     applyRuntime.Cluster,
		Kind:        kind,
		Name:        name,
		ApplyStatus: applyStatus,
	}
	if equality.Semantic.DeepEqual(currIngressApplyConfig, nextIngressApplyConfig) {
		syncStatus = append(syncStatus, s)
		return nil
	}

	applied, err := ingressClient.Apply(
		applyRuntime.Context,
		nextIngressApplyConfig,
		metav1.ApplyOptions{
			FieldManager: fieldMgr,
			Force:        true,
		},
	)
	if err != nil {
		applyStatus = "not applied"
		s.ApplyStatus = applyStatus
		syncStatus = append(syncStatus, s)

		log.Error(err, "unable to apply")
		return err
	}

	syncStatus = append(syncStatus, s)

	log.Info(fmt.Sprintf("Nginx Ingress Applied: [cluster] %s, [resource] %s", applyRuntime.Cluster, applied.GetName()))

	return nil
}

func (r *ReplicatorReconciler) applyIngressSecret(
	applyRuntime ReplicateRuntime,
	fieldMgr string,
) error {
	var (
		log          = applyRuntime.Log
		secretClient = applyRuntime.ClientSet.CoreV1().Secrets(applyRuntime.Replicator.Spec.ReplicationNamespace)
	)

	secret, err := secretClient.Get(
		applyRuntime.Context,
		constants.ClientSecretName,
		metav1.GetOptions{},
	)
	if err != nil {
		// If the resource does not exist, create it.
		// Therefore, Not Found errors are ignored.
		if !errors.IsNotFound(err) {
			return err
		}
	}

	if len(secret.GetName()) > 0 {
		return nil
	}

	caCrt, _, err := pki.CreateCaCrt()
	if err != nil {
		log.Error(err, "Unable create CA Certificates")
		return err
	}

	svrCrt, svrKey, err := pki.CreateSvrCrt(applyRuntime.Replicator)
	if err != nil {
		log.Error(err, "Unable create Server Certificates")
		return err
	}

	secData := map[string][]byte{
		"tls.crt": svrCrt,
		"tls.key": svrKey,
		"ca.crt":  caCrt,
	}

	nextIngressSecretApplyConfig := corev1apply.Secret(
		constants.IngressSecretName,
		applyRuntime.Replicator.Spec.ReplicationNamespace).
		WithData(secData)

	if applyRuntime.IsPrimary {
		owner, err := createOwnerReferences(log, r.Scheme)
		if err != nil {
			log.Error(err, "Unable create OwnerReference")
			return err
		}
		nextIngressSecretApplyConfig.WithOwnerReferences(owner)
	}

	kind := *nextIngressSecretApplyConfig.Kind
	name := *nextIngressSecretApplyConfig.Name
	applyStatus := "applied"
	s := replicatev1.PerResourceApplyStatus{
		Cluster:     applyRuntime.Cluster,
		Kind:        kind,
		Name:        name,
		ApplyStatus: applyStatus,
	}

	applied, err := secretClient.Apply(
		applyRuntime.Context,
		nextIngressSecretApplyConfig,
		metav1.ApplyOptions{
			FieldManager: fieldMgr,
			Force:        true,
		},
	)
	if err != nil {
		applyStatus = "not applied"
		s.ApplyStatus = applyStatus
		syncStatus = append(syncStatus, s)

		log.Error(err, "unable to apply")
		return err
	}

	syncStatus = append(syncStatus, s)

	log.Info(fmt.Sprintf("Nginx Server Certificates Secret Applied: [cluster] %s, [resource] %s", applyRuntime.Cluster, applied.GetName()))

	return nil
}

func (r *ReplicatorReconciler) applyClientSecret(
	applyRuntime ReplicateRuntime,
	fieldMgr string,
) error {
	var (
		log          = applyRuntime.Log
		secretClient = applyRuntime.ClientSet.CoreV1().Secrets(applyRuntime.Replicator.Spec.ReplicationNamespace)
	)

	secret, err := secretClient.Get(
		applyRuntime.Context,
		constants.ClientSecretName,
		metav1.GetOptions{},
	)
	if err != nil {
		// If the resource does not exist, create it.
		// Therefore, Not Found errors are ignored.
		if !errors.IsNotFound(err) {
			return err
		}
	}

	if len(secret.GetName()) > 0 {
		return nil
	}

	cliCrt, cliKey, err := pki.CreateClientCrt()
	if err != nil {
		log.Error(err, "Unable create Client Certificates")
		return err
	}

	secData := map[string][]byte{
		"client.crt": cliCrt,
		"client.key": cliKey,
	}

	nextClientSecretApplyConfig := corev1apply.Secret(
		constants.ClientSecretName,
		applyRuntime.Replicator.Spec.ReplicationNamespace).
		WithData(secData)

	if applyRuntime.IsPrimary {
		owner, err := createOwnerReferences(log, r.Scheme)
		if err != nil {
			log.Error(err, "Unable create OwnerReference")
			return err
		}
		nextClientSecretApplyConfig.WithOwnerReferences(owner)
	}

	applied, err := secretClient.Apply(
		applyRuntime.Context,
		nextClientSecretApplyConfig,
		metav1.ApplyOptions{
			FieldManager: fieldMgr,
			Force:        true,
		},
	)
	if err != nil {
		log.Error(err, "unable to apply")
		return err
	}

	log.Info(fmt.Sprintf("Nginx Client Certificates Secret Applied: [cluster] %s, [resource] %s", applyRuntime.Cluster, applied.GetName()))

	return nil
}

func (r *ReplicatorReconciler) applyResources(
	applyFuncArgs ReplicateRuntime,
) error {
	var applyErr error

	if applyFuncArgs.Replicator.Spec.ConfigMapData != nil {
		// Create Configmap
		// Generate default.conf and index.html
		if err := r.applyConfigMap(
			applyFuncArgs,
			constants.FieldManager,
		); err != nil {
			applyErr = multierr.Append(applyErr, err)
		}
	}

	// Create Deployment
	// Deployment resources are required.
	if err := r.applyDeployment(
		applyFuncArgs,
		constants.FieldManager,
	); err != nil {
		applyErr = multierr.Append(applyErr, err)
	}

	// Create Service
	if applyFuncArgs.Replicator.Spec.ServiceSpec != nil {
		if err := r.applyService(
			applyFuncArgs,
			constants.FieldManager,
		); err != nil {
			applyErr = multierr.Append(applyErr, err)
		}
	}

	// Create Ingress
	if applyFuncArgs.Replicator.Spec.IngressSpec != nil {
		if err := r.applyIngress(
			applyFuncArgs,
			constants.FieldManager,
		); err != nil {
			applyErr = multierr.Append(applyErr, err)
		}
	}

	return applyErr
}

func (r *ReplicatorReconciler) Replicate(
	ctx context.Context,
	log logr.Logger,
	req ctrl.Request,
	primaryCluster map[string]*kubernetes.Clientset,
	secondaryClientSets map[string]*kubernetes.Clientset,
) error {
	var (
		applyFailed      bool
		err              error
		replicateRuntime ReplicateRuntime
		clusterDetectors replicatev1.ClusterDetectorList
	)
	if err := r.Client.List(ctx, &clusterDetectors, client.InNamespace(constants.Namespace)); err != nil {
		return err
	}

	for primaryClusterName, clientSet := range primaryCluster {
		replicateRuntime = ReplicateRuntime{
			ClientSet:  clientSet,
			IsPrimary:  true,
			Context:    ctx,
			Log:        log,
			Cluster:    primaryClusterName,
			Replicator: replicator,
			Request:    req,
		}
		if err := r.applyResources(replicateRuntime); err != nil {
			log.Error(err, fmt.Sprintf("Could not apply to Primary Cluster %s", primaryClusterName))
			return err
		}
	}

	// After successful resource deployment to the local cluster,
	// replicate the resources to the remote cluster.
	for secondaryClusterName, clientSet := range secondaryClientSets {
		replicateRuntime.ClientSet = clientSet
		replicateRuntime.IsPrimary = false
		replicateRuntime.Cluster = secondaryClusterName
		if err = r.applyResources(replicateRuntime); err != nil {
			applyFailed = true
			log.Error(err, fmt.Sprintf("Could not replicate to Secondary Cluster %s", secondaryClusterName))
		}
	}
	// If one of the clusters fails to replicate, it is considered a synchronization failure.
	if applyFailed {
		log.Error(err, "Could not sync on all clusters")
		return err
	}

	return nil
}

func (r *ReplicatorReconciler) createNamespace(
	ctx context.Context,
	log logr.Logger,
	replicator replicatev1.Replicator,
	secondaryClientSets map[string]*kubernetes.Clientset,
) error {
	for cluster, clientSet := range secondaryClientSets {
		var namespaceClient = clientSet.CoreV1().Namespaces()

		ns := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: replicator.Spec.ReplicationNamespace,
			},
		}

		if _, err := namespaceClient.Get(
			ctx,
			replicator.Spec.ReplicationNamespace,
			metav1.GetOptions{},
		); err != nil {
			// If the resource does not exist, create it.
			// Therefore, Not Found errors are ignored.
			if !errors.IsNotFound(err) {
				return err
			}

			created, err := namespaceClient.Create(ctx, ns, metav1.CreateOptions{})
			if err != nil {
				log.Error(err, "unable create namespace")
				return err
			}

			log.Info(fmt.Sprintf("Namespace creation: [cluster] %s, [resource] %s", cluster, created.GetName()))
		}
	}
	return nil
}

func deletePrimaryNamespace(
	ctx context.Context,
	log logr.Logger,
	primaryClientSet map[string]*kubernetes.Clientset,
	primaryNamespaceName string,
) error {
	for _, clientSet := range primaryClientSet {
		var namespaceClient = clientSet.CoreV1().Namespaces()
		if err := namespaceClient.Delete(
			ctx,
			primaryNamespaceName,
			metav1.DeleteOptions{},
		); err != nil {
			log.Error(err, "unable to delete primary namespace")
		}
	}

	return nil
}

func deleteSecondaryClusterResources(
	ctx context.Context,
	replicator replicatev1.Replicator,
	secondaryClientSets map[string]*kubernetes.Clientset,
) error {
	for _, clientSet := range secondaryClientSets {
		var (
			configMapClient  = clientSet.CoreV1().ConfigMaps(replicator.Spec.ReplicationNamespace)
			deploymentClient = clientSet.AppsV1().Deployments(replicator.Spec.ReplicationNamespace)
			serviceClient    = clientSet.CoreV1().Services(replicator.Spec.ReplicationNamespace)
			ingressClient    = clientSet.NetworkingV1().Ingresses(replicator.Spec.ReplicationNamespace)
			secretClient     = clientSet.CoreV1().Secrets(replicator.Spec.ReplicationNamespace)
			namespaceClient  = clientSet.CoreV1().Namespaces()
		)

		if replicator.Spec.IngressSecureEnabled {
			if err := secretClient.Delete(
				ctx,
				constants.ClientSecretName,
				metav1.DeleteOptions{},
			); err != nil {
				return err
			}

			if err := secretClient.Delete(
				ctx,
				constants.IngressSecretName,
				metav1.DeleteOptions{},
			); err != nil {
				return err
			}
		}

		if replicator.Spec.IngressSpec != nil {
			if err := ingressClient.Delete(
				ctx,
				replicator.Spec.IngressName,
				metav1.DeleteOptions{},
			); err != nil {
				return nil
			}
		}

		if replicator.Spec.ServiceSpec != nil {
			if err := serviceClient.Delete(
				ctx,
				replicator.Spec.ServiceName,
				metav1.DeleteOptions{},
			); err != nil {
				return err
			}
		}

		// Required Resources
		if err := deploymentClient.Delete(
			ctx,
			replicator.Spec.DeploymentName,
			metav1.DeleteOptions{},
		); err != nil {
			return err
		}

		if replicator.Spec.ConfigMapData != nil {
			if err := configMapClient.Delete(
				ctx,
				replicator.Spec.ConfigMapName,
				metav1.DeleteOptions{},
			); err != nil {
				return err
			}
		}

		if err := namespaceClient.Delete(
			ctx,
			replicator.Spec.ReplicationNamespace,
			metav1.DeleteOptions{},
		); err != nil {
			return err
		}
	}

	return nil
}

//+kubebuilder:rbac:groups=replicate.jnytnai0613.github.io,resources=replicators,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=replicate.jnytnai0613.github.io,resources=replicators/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=replicate.jnytnai0613.github.io,resources=replicators/finalizers,verbs=update
//+kubebuilder:rbac:groups=core,resources=namespaces,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=endpoints,verbs=get;list;watch
//+kubebuilder:rbac:groups=networking.k8s.io,resources=ingresses,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete

// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.6/pkg/reconcile
func (r *ReplicatorReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var (
		applyFailed          bool
		logger               = log.FromContext(ctx)
		secondaryClientSets  = make(map[string]*kubernetes.Clientset)
		clusterDetectors     replicatev1.ClusterDetectorList
		primaryClientSet     = make(map[string]*kubernetes.Clientset)
		primaryNamespaceName string
	)
	if err := r.Client.List(ctx, &clusterDetectors, client.InNamespace(constants.Namespace)); err != nil {
		return ctrl.Result{}, err
	}

	if err := r.Client.Get(ctx, req.NamespacedName, &replicator); err != nil {
		logger.Error(err, "unable to fetch CR Replicator")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Generate ClientSet for primary cluster.
	for _, clusterDetector := range clusterDetectors.Items {
		if clusterDetector.GetLabels()["app.kubernetes.io/role"] != "primary" {
			continue
		}
		primaryClusterName := clusterDetector.GetName()
		for key, clientSet := range r.ClientSets {
			if key != primaryClusterName {
				continue
			}
			primaryClientSet[primaryClusterName] = clientSet
		}
	}

	// Generate ClientSet for secondary cluster.
	// ClientSet for secondary cluster are used for replication and Finalize.
	for key, clientSet := range r.ClientSets {
		for _, secondaryCluster := range replicator.Spec.TargetCluster {
			if key != secondaryCluster {
				continue
			}
			secondaryClientSets[secondaryCluster] = clientSet
			break
		}
	}

	// Resources in secondary clusters are considered external resources.
	// Therefore, they are deleted by finalizer.
	finalizerName := "replicate.jnytnai0613.github.io/finalizer"
	if !replicator.ObjectMeta.DeletionTimestamp.IsZero() {
		primaryNamespaceName = replicator.Spec.ReplicationNamespace
		if controllerutil.ContainsFinalizer(&replicator, finalizerName) {
			if err := deleteSecondaryClusterResources(ctx, replicator, secondaryClientSets); err != nil {
				return ctrl.Result{}, err
			}

			controllerutil.RemoveFinalizer(&replicator, finalizerName)
			if err := r.Update(ctx, &replicator); err != nil {
				return ctrl.Result{}, err
			}
		}

		// After controllerutil.RemoveFinalizer processing, the Replicator resource is deleted.
		// Since the child resources are deleted by deleting the Replicator resource,
		// namespace can also be deleted.
		if err := deletePrimaryNamespace(
			ctx,
			logger,
			primaryClientSet,
			primaryNamespaceName,
		); err != nil {
			return ctrl.Result{}, err
		}

		return ctrl.Result{}, nil
	}

	if !controllerutil.ContainsFinalizer(&replicator, finalizerName) {
		controllerutil.AddFinalizer(&replicator, finalizerName)
		if err := r.Update(ctx, &replicator); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Create a namespace for replication in the Primary Cluster.
	if err := r.createNamespace(ctx, logger, replicator, primaryClientSet); err != nil {
		return ctrl.Result{}, err
	}

	// Create a namespace for replication in the Secondary Cluster.
	if err := r.createNamespace(ctx, logger, replicator, secondaryClientSets); err != nil {
		return ctrl.Result{}, err
	}

	// Initialize syncStatus slice once to update the status of the replicator.
	// If not initialized, the status held in the previous Reconcile is used.
	syncStatus = nil
	err := r.Replicate(ctx, logger, req, primaryClientSet, secondaryClientSets)

	replicator.Status.Applied = syncStatus
	for _, v := range replicator.Status.Applied {
		if v.ApplyStatus == "not applied" {
			applyFailed = true
			replicator.Status.Synced = "not synced"
			if err := r.Status().Update(ctx, &replicator); err != nil {
				return ctrl.Result{}, err
			}
		}
	}

	// Handle errors in the r.Replicate function after Status
	// processing of the replicator resource.
	if applyFailed {
		return ctrl.Result{}, err
	}

	replicator.Status.Applied = syncStatus
	replicator.Status.Synced = "synced"
	if err := r.Status().Update(ctx, &replicator); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ReplicatorReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&replicatev1.Replicator{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Owns(&networkv1.Ingress{}).
		Complete(r)
}
