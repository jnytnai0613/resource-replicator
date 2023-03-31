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

package main

import (
	"flag"
	"os"
	"time"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	replicatev1 "github.com/jnytnai0613/resource-replicator/api/v1"
	"github.com/jnytnai0613/resource-replicator/controllers"
	cli "github.com/jnytnai0613/resource-replicator/pkg/client"
	//+kubebuilder:scaffold:imports
)

var (
	LocalClient          client.Client
	enableLeaderElection bool

	metricsAddr string
	probeAddr   string
	scheme      = runtime.NewScheme()
	setupLog    = ctrl.Log.WithName("setup")
	clientSets  map[string]*kubernetes.Clientset
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(replicatev1.AddToScheme(scheme))

	//+kubebuilder:scaffold:scheme

	flag.StringVar(
		&metricsAddr,
		"metrics-bind-address",
		":8080",
		"The address the metric endpoint binds to.",
	)
	flag.StringVar(
		&probeAddr,
		"health-probe-bind-address",
		":8081",
		"The address the probe endpoint binds to.",
	)
	flag.BoolVar(
		&enableLeaderElection,
		"leader-elect",
		false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.",
	)
	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	// Use the credentials in the Controller Pod's ServiceAccount to generate clients.
	// Used for the following purposes
	// - Initialization of ClusterDetector resource
	// - Obtaining the Secret resource object when extracting kubeconfig
	//   from the config Secret resource in the kubeconfig namespace
	localClient, _ := cli.CreateLocalClient(setupLog, *scheme)

	// Initialization of ClusterDetector resource
	err := controllers.SetupClusterDetector(localClient, setupLog)
	if err != nil {
		setupLog.Error(err, "Failed to initialize ClusterDetector.")
		os.Exit(1)
	}
	setupLog.Info("Initialization of all ClusterDetector resources completed")

	// Extract context information from kubeconfig file and
	// generate clientsets for each context
	clientSets, err = cli.CreateClientSets(localClient)
	if err != nil {
		setupLog.Error(err, "Failed to create remote clientset.")
	}
}

func main() {
	var resyncPeriod = time.Second * 30

	mgr, err := ctrl.NewManager(
		ctrl.GetConfigOrDie(),
		ctrl.Options{
			Scheme:                 scheme,
			SyncPeriod:             &resyncPeriod,
			MetricsBindAddress:     metricsAddr,
			Port:                   9443,
			HealthProbeBindAddress: probeAddr,
			LeaderElection:         enableLeaderElection,
			LeaderElectionID:       "90fe7186.jnytnai0613.github.io",
			// LeaderElectionReleaseOnCancel defines if the leader should step down voluntarily
			// when the Manager ends. This requires the binary to immediately end when the
			// Manager is stopped, otherwise, this setting is unsafe. Setting this significantly
			// speeds up voluntary leader transitions as the new leader don't have to wait
			// LeaseDuration time first.
			//
			// In the default scaffold provided, the program ends immediately after
			// the manager stops, so would be fine to enable this option. However,
			// if you are doing or is intended to do any operation such as perform cleanups
			// after the manager stops then its usage might be unsafe.
			// LeaderElectionReleaseOnCancel: true,
		},
	)
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	if err = (&controllers.ClusterDetectorReconciler{
		Client: mgr.GetClient(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err,
			"unable to create controller",
			"controller",
			"ClusterDetector",
		)
		os.Exit(1)
	}
	if err = (&controllers.ReplicatorReconciler{
		Client:     mgr.GetClient(),
		ClientSets: clientSets,
		Recorder:   mgr.GetEventRecorderFor("Replicator"),
		Scheme:     mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Replicator")
		os.Exit(1)
	}
	//+kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
