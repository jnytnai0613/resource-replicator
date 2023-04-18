package controllers

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	appsv1apply "k8s.io/client-go/applyconfigurations/apps/v1"
	corev1apply "k8s.io/client-go/applyconfigurations/core/v1"
	networkv1apply "k8s.io/client-go/applyconfigurations/networking/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	replicatev1 "github.com/jnytnai0613/resource-replicator/api/v1"
	"github.com/jnytnai0613/resource-replicator/pkg/constants"
)

var (
	cip                = corev1.ServiceTypeClusterIP
	defaultconf string = `server {
	listen 80 default_server;
	listen [::]:80 default_server ipv6only=on;
	root /usr/share/nginx/html;
	index index.html index.htm mod-index.html;
	server_name localhost;
}`
	indexhtml = `<!DOCTYPE html>
<html>
<head>
<title>Yeahhhhhhh!! Welcome to nginx!!</title>
<style>
html { color-scheme: light dark; }
body { width: 35em; margin: 0 auto;
font-family: Tahoma, Verdana, Arial, sans-serif; }
</style>
</head>
<body>
<h1>Yeahhhhhhh!! Welcome to nginx!!</h1>
<p>If you see this page, the nginx web server is successfully installed and
working. Further configuration is required.</p>
<p>For online documentation and support please refer to
<a href="http://nginx.org/">nginx.org</a>.<br/>
Commercial support is available at
<a href="http://nginx.com/">nginx.com</a>.</p>
<p><em>Thank you for using nginx.</em></p>
</body>
</html>`
	hostname                   = "nginx.example.com"
	image                      = "nginx"
	port                       = 80
	replicationNamespace       = "test-ns"
	resouceName                = "nginx"
	rval                 int32 = 3
)

func testReplicator() *replicatev1.Replicator {
	replicator := &replicatev1.Replicator{}
	replicator.Namespace = constants.Namespace
	replicator.Name = "test"

	replicator.Spec.ConfigMapName = resouceName
	m := make(map[string]string)
	m["default.conf"] = defaultconf
	m["index.html"] = indexhtml
	replicator.Spec.ConfigMapData = m

	replicator.Spec.ReplicationNamespace = replicationNamespace
	replicator.Spec.TargetCluster = append(replicator.Spec.TargetCluster, "cluster1")
	replicator.Spec.TargetCluster = append(replicator.Spec.TargetCluster, "cluster2")
	replicator.Spec.DeploymentName = resouceName
	depSpec := appsv1apply.DeploymentSpec()
	depSpec.WithReplicas(rval).
		WithTemplate(corev1apply.PodTemplateSpec().
			WithSpec(corev1apply.PodSpec().
				WithContainers(corev1apply.Container().
					WithName(resouceName).
					WithImage(image))))
	replicator.Spec.DeploymentSpec = (*replicatev1.DeploymentSpecApplyConfiguration)(depSpec)

	replicator.Spec.ServiceName = resouceName
	svcSpec := corev1apply.ServiceSpec()
	svcSpec.WithType(cip).
		WithPorts(corev1apply.ServicePort().
			WithProtocol(corev1.ProtocolTCP).
			WithPort(int32(port)).
			WithTargetPort(intstr.FromInt(port)))
	replicator.Spec.ServiceSpec = (*replicatev1.ServiceSpecApplyConfiguration)(svcSpec)

	replicator.Spec.IngressName = resouceName
	ingressspec := networkv1apply.IngressSpec()
	ingressspec.WithIngressClassName(constants.IngressClassName).
		WithRules(networkv1apply.IngressRule().
			WithHost(hostname).
			WithHTTP(networkv1apply.HTTPIngressRuleValue().
				WithPaths(networkv1apply.HTTPIngressPath().
					WithPath("/").
					WithPathType(networkingv1.PathTypePrefix).
					WithBackend(networkv1apply.IngressBackend().
						WithService(networkv1apply.IngressServiceBackend().
							WithName("nginx").
							WithPort(networkv1apply.ServiceBackendPort().
								WithNumber(int32(port))))))))
	replicator.Spec.IngressSpec = (*replicatev1.IngressSpecApplyConfiguration)(ingressspec)
	replicator.Spec.IngressSecureEnabled = false

	return replicator
}

var err error

var _ = Describe("Test Controller", func() {
	ctx := context.Background()
	var stopFunc func()

	BeforeEach(func() {
		mgr, err := ctrl.NewManager(cfgs[0], ctrl.Options{
			Scheme:             scheme,
			LeaderElection:     false,
			MetricsBindAddress: "0",
		})
		Expect(err).ShouldNot(HaveOccurred())

		reconciler := &ReplicatorReconciler{
			Client:     mgr.GetClient(),
			ClientSets: clientsets,
			Recorder:   mgr.GetEventRecorderFor("replicator-controller"),
			Scheme:     scheme,
		}
		err = reconciler.SetupWithManager(mgr)
		Expect(err).ShouldNot(HaveOccurred())

		ctx, cancel := context.WithCancel(ctx)
		stopFunc = cancel
		go func() {
			err := mgr.Start(ctx)
			if err != nil {
				panic(err)
			}
		}()
		time.Sleep(100 * time.Millisecond)
	})

	AfterEach(func() {
		stopFunc()
		time.Sleep(100 * time.Millisecond)
	})

	It("should create custom resource", func() {
		cr := testReplicator()
		err = kClient.Create(ctx, cr)
		Expect(err).ShouldNot(HaveOccurred())

		Eventually(func(g Gomega) {
			cr := &replicatev1.Replicator{}
			key := client.ObjectKey{Namespace: constants.Namespace, Name: "test"}
			err := kClient.Get(ctx, key, cr)
			g.Expect(err).ShouldNot(HaveOccurred())
		}).Should(Succeed())
	})

	It("should create configmap resource", func() {
		configMapClient := clientsets["cluster1"].CoreV1().ConfigMaps(replicationNamespace)
		cm := &corev1.ConfigMap{}
		Eventually(func(g Gomega) {
			cm, err = configMapClient.Get(ctx, resouceName, metav1.GetOptions{})
			g.Expect(err).ShouldNot(HaveOccurred())
		}, 10*time.Second).Should(Succeed())

		Expect(cm.Data["default.conf"]).Should(Equal(defaultconf))
		Expect(cm.Data["index.html"]).Should(Equal(indexhtml))
	})

	It("should create deployment resource", func() {
		deploymentClient := clientsets["cluster1"].AppsV1().Deployments(replicationNamespace)
		dep := &appsv1.Deployment{}
		Eventually(func(g Gomega) {
			dep, err = deploymentClient.Get(ctx, resouceName, metav1.GetOptions{})
			g.Expect(err).ShouldNot(HaveOccurred())
		}, 10*time.Second).Should(Succeed())

		var r int32 = rval
		Expect(dep.Spec.Replicas).Should(Equal(&r))
		Expect(dep.Spec.Template.Spec.Containers[0].Name).Should(Equal(resouceName))
		Expect(dep.Spec.Template.Spec.Containers[0].Image).Should(Equal(image))
	})

	It("should create service resource", func() {
		serviceClient := clientsets["cluster1"].CoreV1().Services(replicationNamespace)
		svc := &corev1.Service{}
		Eventually(func(g Gomega) {
			svc, err = serviceClient.Get(ctx, resouceName, metav1.GetOptions{})
			g.Expect(err).ShouldNot(HaveOccurred())
		}, 10*time.Second).Should(Succeed())

		Expect(svc.Spec.Type).Should(Equal(cip))
		Expect(svc.Spec.Ports[0].Protocol).Should(Equal(corev1.ProtocolTCP))
		Expect(svc.Spec.Ports[0].Port).Should(Equal(int32(port)))
		Expect(svc.Spec.Ports[0].TargetPort).Should(Equal(intstr.FromInt(port)))
	})

	It("should create ingress resource", func() {
		ingressClient := clientsets["cluster1"].NetworkingV1().Ingresses(replicationNamespace)
		ing := &networkingv1.Ingress{}
		Eventually(func(g Gomega) {
			ing, err = ingressClient.Get(ctx, resouceName, metav1.GetOptions{})
			g.Expect(err).ShouldNot(HaveOccurred())
		}, 5*time.Second).Should(Succeed())

		bport := networkingv1.ServiceBackendPort{
			Number: int32(port),
		}
		Expect(*ing.Spec.IngressClassName).Should(Equal(constants.IngressClassName))
		Expect(ing.Spec.Rules[0].Host).Should(Equal(hostname))
		Expect(ing.Spec.Rules[0].HTTP.Paths[0].Path).Should(Equal("/"))
		Expect(*ing.Spec.Rules[0].HTTP.Paths[0].PathType).Should(Equal(networkingv1.PathTypePrefix))
		Expect(ing.Spec.Rules[0].HTTP.Paths[0].Backend.Service.Name).Should(Equal("nginx"))
		Expect(ing.Spec.Rules[0].HTTP.Paths[0].Backend.Service.Port).Should(Equal(bport))
	})

	It("should create ca/server/client certtificate secret resource", func() {
		cr := &replicatev1.Replicator{}
		key := client.ObjectKey{Namespace: constants.Namespace, Name: "test"}
		err := kClient.Get(ctx, key, cr)
		Expect(err).ShouldNot(HaveOccurred())

		cr.Spec.IngressSecureEnabled = true
		err = kClient.Update(ctx, cr)
		Expect(err).ShouldNot(HaveOccurred())

		secretClient := clientsets["cluster1"].CoreV1().Secrets(replicationNamespace)
		caSec := &corev1.Secret{}
		Eventually(func(g Gomega) {
			caSec, err = secretClient.Get(ctx, constants.IngressSecretName, metav1.GetOptions{})
			g.Expect(err).ShouldNot(HaveOccurred())
		}, 5*time.Second).Should(Succeed())

		Eventually(func(g Gomega) {
			_, err = secretClient.Get(ctx, constants.ClientSecretName, metav1.GetOptions{})
			g.Expect(err).ShouldNot(HaveOccurred())
		}, 5*time.Second).Should(Succeed())

		ingressClient := clientsets["cluster1"].NetworkingV1().Ingresses(replicationNamespace)
		ing := &networkingv1.Ingress{}
		Eventually(func(g Gomega) {
			ing, err = ingressClient.Get(ctx, resouceName, metav1.GetOptions{})
			Expect(err).ShouldNot(HaveOccurred())
		}, 5*time.Second).Should(Succeed())

		Expect(ing.Spec.TLS[0].Hosts[0]).Should(Equal(hostname))
		Expect(ing.Spec.TLS[0].SecretName).Should(Equal(caSec.GetName()))
	})

})
