/*

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
	"crypto/tls"
	"flag"
	"os"

	certmanager "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	stepv1beta1 "github.com/smallstep/step-issuer/api/v1beta1"
	"github.com/smallstep/step-issuer/controllers"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/utils/clock"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/metrics/filters"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	// +kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	_ = clientgoscheme.AddToScheme(scheme)
	_ = certmanager.AddToScheme(scheme)
	_ = stepv1beta1.AddToScheme(scheme)
	// +kubebuilder:scaffold:scheme
}

func main() {
	var metricsAddr string
	var secureMetrics bool
	var disableHTTP2 bool
	var enableLeaderElection bool
	var leaderElectionID string
	var disableApprovedCheck bool

	// Options for configuring logging
	opts := zap.Options{}
	opts.BindFlags(flag.CommandLine)

	flag.StringVar(&metricsAddr, "metrics-bind-address", "0",
		"The address the metrics endpoint binds to. Use :8443 for HTTPS or :8080 for HTTP, or leave as 0 to disable the metrics service.")
	flag.BoolVar(&secureMetrics, "metrics-secure", true,
		"If set, the metrics endpoint is served securely via HTTPS. Use --metrics-secure=false to use HTTP instead.")
	flag.BoolVar(&disableHTTP2, "disable-http2", false,
		"If set, HTTP/2 will be disabled for the metrics server, mitigating CVE-2023-44487 and CVE-2023-39325.")
	flag.BoolVar(&enableLeaderElection, "enable-leader-election", false,
		"Enable leader election for controller manager. Enabling this will ensure there is only one active controller manager.")
	flag.StringVar(&leaderElectionID, "leader-election-id", "",
		"The name of the resource that leader election will use for holding the leader lock.")
	flag.BoolVar(&disableApprovedCheck, "disable-approval-check", false,
		"Disables waiting for CertificateRequests to have an approved condition before signing.")
	flag.Parse()

	if enableLeaderElection && leaderElectionID == "" {
		leaderElectionID = "step-issuer-operator-lock"
	}

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	var tlsOpts []func(*tls.Config)
	if disableHTTP2 {
		tlsOpts = append(tlsOpts, func(c *tls.Config) {
			setupLog.Info("disabling http/2")
			c.NextProtos = []string{"http/1.1"}
		})
	}

	metricsServerOptions := metricsserver.Options{
		BindAddress:   metricsAddr,
		SecureServing: secureMetrics,
		TLSOpts:       tlsOpts,
	}
	if secureMetrics {
		metricsServerOptions.FilterProvider = filters.WithAuthenticationAndAuthorization
	}
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:           scheme,
		Metrics:          metricsServerOptions,
		LeaderElection:   enableLeaderElection,
		LeaderElectionID: leaderElectionID,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	if err = (&controllers.StepIssuerReconciler{
		Client:   mgr.GetClient(),
		Log:      ctrl.Log.WithName("controllers").WithName("StepIssuer"),
		Clock:    clock.RealClock{},
		Recorder: mgr.GetEventRecorderFor("stepissuer-controller"), //nolint:staticcheck,nolintlint // will be fixed later
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "StepIssuer")
		os.Exit(1)
	}

	if err = (&controllers.StepClusterIssuerReconciler{
		Client:   mgr.GetClient(),
		Log:      ctrl.Log.WithName("controllers").WithName("StepClusterIssuer"),
		Clock:    clock.RealClock{},
		Recorder: mgr.GetEventRecorderFor("stepclusterissuer-controller"), //nolint:staticcheck,nolintlint // will be fixed later
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "StepClusterIssuer")
		os.Exit(1)
	}

	if err = (&controllers.CertificateRequestReconciler{
		Client:                 mgr.GetClient(),
		Log:                    ctrl.Log.WithName("controllers").WithName("CertificateRequest"),
		Recorder:               mgr.GetEventRecorderFor("certificaterequests-controller"), //nolint:staticcheck,nolintlint // will be fixed later
		Clock:                  clock.RealClock{},
		CheckApprovedCondition: !disableApprovedCheck,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "CertificateRequest")
		os.Exit(1)
	}

	// +kubebuilder:scaffold:builder

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
