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

package controllers

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	api "github.com/smallstep/step-issuer/api/v1beta1"
	"github.com/smallstep/step-issuer/provisioners"
	"k8s.io/client-go/tools/record"
	"k8s.io/utils/clock"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// StepClusterIssuerReconciler reconciles a StepClusterIssuer object
type StepClusterIssuerReconciler struct {
	client.Client
	Log      logr.Logger
	Clock    clock.Clock
	Recorder record.EventRecorder
}

// +kubebuilder:rbac:groups=certmanager.step.sm,resources=stepclusterissuers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=certmanager.step.sm,resources=stepclusterissuers/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch
// +kubebuilder:rbac:groups=coordination.k8s.io,resources=leases,verbs=get;create;update

// Reconcile will read and validate the StepClusterIssuer resources, it will set the
// status condition ready to true if everything is right.
func (r *StepClusterIssuerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("stepclusterissuer", req.NamespacedName)

	iss := new(api.StepClusterIssuer)
	if err := r.Client.Get(ctx, req.NamespacedName, iss); err != nil {
		log.Error(err, "failed to retrieve StepClusterIssuer resource")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	statusReconciler := newStepStatusClusterReconciler(r, iss, log)
	if err := validateStepClusterIssuerSpec(iss.Spec); err != nil {
		log.Error(err, "failed to validate StepClusterIssuer resource")
		statusReconciler.UpdateNoError(ctx, api.ConditionFalse, "Validation", "Failed to validate resource: %v", err)
		return ctrl.Result{}, err
	}

	// Fetch the provisioner password from the configured source: a Kubernetes
	// Secret, an environment variable, or a file on the controller's filesystem.
	password, notFound, err := resolveProvisionerPassword(ctx, r.Client, iss.Spec.Provisioner.PasswordRef.Namespace,
		iss.Spec.Provisioner.PasswordRef.Name, iss.Spec.Provisioner.PasswordRef.Key,
		iss.Spec.Provisioner.PasswordEnv, iss.Spec.Provisioner.PasswordFile)
	if err != nil {
		log.Error(err, "failed to retrieve StepClusterIssuer provisioner password", "name", req.Name)
		reason := "Error"
		if notFound {
			reason = "NotFound"
		}
		statusReconciler.UpdateNoError(ctx, api.ConditionFalse, reason, "Failed to retrieve provisioner password: %v", err)
		return ctrl.Result{}, err
	}

	//Verify that the CABundle is in x509 PEM format If not then covert it over
	//to PEM x509 format.
	if !isPEMFormat(iss.Spec.CABundle) {
		caBundle, err := convertToPemFormat(iss.Spec.CABundle)
		if err != nil {
			log.Error(err, "failed to parse caBundle in the StepClusterIssuer spec", "name", req.Name)
		}
		iss.Spec.CABundle = caBundle
	}

	// Initialize and store the provisioner
	p, err := provisioners.NewFromStepClusterIssuer(iss, password)
	if err != nil {
		log.Error(err, "failed to initialize provisioner")
		statusReconciler.UpdateNoError(ctx, api.ConditionFalse, "Error", "failed initialize provisioner")
		return ctrl.Result{}, err
	}
	provisioners.Store(req.NamespacedName, p)

	return ctrl.Result{}, statusReconciler.Update(ctx, api.ConditionTrue, "Verified", "StepClusterIssuer verified and ready to sign certificates")
}

// SetupWithManager initializes the StepClusterIssuer controller into the controller
// runtime.
func (r *StepClusterIssuerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&api.StepClusterIssuer{}).
		Complete(r)
}

func validateStepClusterIssuerSpec(s api.StepClusterIssuerSpec) error {
	switch {
	case s.URL == "":
		return fmt.Errorf("spec.url cannot be empty")
	case len(s.CABundle) == 0:
		return fmt.Errorf("spec.caBundle cannot be empty")
	case s.Provisioner.Name == "":
		return fmt.Errorf("spec.provisioner.name cannot be empty")
	case s.Provisioner.KeyID == "":
		return fmt.Errorf("spec.provisioner.kid cannot be empty")
	default:
		return validateProvisionerPasswordSource(s.Provisioner.PasswordRef.Name, s.Provisioner.PasswordRef.Key, s.Provisioner.PasswordEnv, s.Provisioner.PasswordFile)
	}
}
