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
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	//+kubebuilder:scaffold:imports

	replicatev1 "github.com/jnytnai0613/resource-replicator/api/v1"
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var (
	cfg        *rest.Config
	cfgs       []*rest.Config
	ctx        = context.Background()
	envs       []*envtest.Environment
	kClient    client.Client
	clientsets = make(map[string]*kubernetes.Clientset)
	scheme     = runtime.NewScheme()
)

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "Controller Suite")
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	numClusters := 2

	err = clientgoscheme.AddToScheme(scheme)
	Expect(err).NotTo(HaveOccurred())
	err = replicatev1.AddToScheme(scheme)
	Expect(err).NotTo(HaveOccurred())

	// start the environments
	for i := 0; i < numClusters; i++ {
		env := &envtest.Environment{
			CRDDirectoryPaths: []string{filepath.Join("..", "config", "crd", "bases")},
			CRDInstallOptions: envtest.CRDInstallOptions{
				Scheme: scheme,
			},
			ErrorIfCRDPathMissing: true,
		}
		envs = append(envs, env)

		cfg, err = env.Start()
		Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("Failed to start env %d", i))

		cfgs = append(cfgs, cfg)

		// Assuming cluster0 is the primary, generate clients.
		if i == 0 {
			kClient, err = client.New(cfg, client.Options{Scheme: scheme})
			Expect(err).NotTo(HaveOccurred())
			Expect(kClient).NotTo(BeNil())
		}

		//+kubebuilder:scaffold:scheme

		// Create clientset for the environment
		kClientset, err := kubernetes.NewForConfig(cfg)
		Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("Failed to create clientset for env %d", i))

		clientsets[fmt.Sprintf("Cluster%d", i)] = kClientset

		ns := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: replicationNamespace,
			},
		}

		clientsets[fmt.Sprintf("cluster%d", i)] = kClientset
		namespaceClient := clientsets[fmt.Sprintf("Cluster%d", i)].CoreV1().Namespaces()
		_, err = namespaceClient.Create(ctx, ns, metav1.CreateOptions{})
		Expect(err).ShouldNot(HaveOccurred())
	}
})

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	for _, env := range envs {
		err := env.Stop()
		Expect(err).NotTo(HaveOccurred())
	}
})
