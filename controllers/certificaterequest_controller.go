/*
Copyright 2019 The cert-manager authors.

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
	apiutil "github.com/jetstack/cert-manager/pkg/api/util"
	cmapi "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1"
	cmmeta "github.com/jetstack/cert-manager/pkg/apis/meta/v1"
	api "github.com/smallstep/step-issuer/api/v1beta1"
	"github.com/smallstep/step-issuer/provisioners"
	core "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"k8s.io/utils/clock"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// CertificateRequestReconciler reconciles a StepIssuer object.
type CertificateRequestReconciler struct {
	client.Client
	Log      logr.Logger
	Recorder record.EventRecorder

	Clock                  clock.Clock
	CheckApprovedCondition bool
}

// +kubebuilder:rbac:groups=cert-manager.io,resources=certificaterequests,verbs=get;list;watch;update
// +kubebuilder:rbac:groups=cert-manager.io,resources=certificaterequests/status,verbs=get;update;patch

// Reconcile will read and validate a StepIssuer resource associated to the
// CertificateRequest resource, and it will sign the CertificateRequest with the
// provisioner in the StepIssuer.
func (r *CertificateRequestReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("certificaterequest", req.NamespacedName)

	// Fetch the CertificateRequest resource being reconciled.
	// Just ignore the request if the certificate request has been deleted.
	cr := new(cmapi.CertificateRequest)
	if err := r.Client.Get(ctx, req.NamespacedName, cr); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}

		log.Error(err, "failed to retrieve CertificateRequest resource")
		return ctrl.Result{}, err
	}

	// Check the CertificateRequest's issuerRef and if it does not match the api
	// group name, log a message at a debug level and stop processing.
	if cr.Spec.IssuerRef.Group != "" && cr.Spec.IssuerRef.Group != api.GroupVersion.Group {
		log.V(4).Info("resource does not specify an issuerRef group name that we are responsible for", "group", cr.Spec.IssuerRef.Group)
		return ctrl.Result{}, nil
	}

	// If CertificateRequest has been denied, mark the CertificateRequest as
	// Ready=Denied and set FailureTime if not already.
	if apiutil.CertificateRequestIsDenied(cr) {
		log.V(4).Info("CertificateRequest has been denied yet. Marking as failed.")

		if cr.Status.FailureTime == nil {
			nowTime := metav1.NewTime(r.Clock.Now())
			cr.Status.FailureTime = &nowTime
		}

		message := "The CertificateRequest was denied by an approval controller"
		return ctrl.Result{}, r.setStatus(ctx, cr, cmmeta.ConditionFalse, cmapi.CertificateRequestReasonDenied, message)
	}

	if r.CheckApprovedCondition {
		// If CertificateRequest has not been approved, exit early.
		if !apiutil.CertificateRequestIsApproved(cr) {
			log.V(4).Info("certificate request has not been approved yet, ignoring")
			return ctrl.Result{}, nil
		}
	}

	// If the certificate data is already set then we skip this request as it
	// has already been completed in the past.
	if len(cr.Status.Certificate) > 0 {
		log.V(4).Info("existing certificate data found in status, skipping already completed CertificateRequest")
		return ctrl.Result{}, nil
	}

	// Step CA does not support online signing of CA certificate at this time
	if cr.Spec.IsCA {
		log.Info("step certificate does not support online signing of CA certificates")
		return ctrl.Result{}, nil
	}

	if cr.Spec.IssuerRef.Kind == "StepClusterIssuer" {
		iss := api.StepClusterIssuer{}
		issNamespaceName := types.NamespacedName{
			Namespace: "",
			Name: cr.Spec.IssuerRef.Name,
		}

		if err := r.Client.Get(ctx, issNamespaceName, &iss); err != nil {
			log.Error(err, "failed to retrieve StepClusterIssuer resource", "namespace", req.Namespace, "name", cr.Spec.IssuerRef.Name)
			_ = r.setStatus(ctx, cr, cmmeta.ConditionFalse, cmapi.CertificateRequestReasonPending, "Failed to retrieve StepClusterIssuer resource %s: %v", issNamespaceName, err)
			return ctrl.Result{}, err
		}
	
		// Check if the StepClusterIssuer resource has been marked Ready
		if !stepClusterIssuerHasCondition(iss, api.StepClusterIssuerCondition{Type: api.ConditionReady, Status: api.ConditionTrue}) {
			err := fmt.Errorf("resource %s is not ready", issNamespaceName)
			log.Error(err, "failed to retrieve StepClusterIssuer resource", "namespace", req.Namespace, "name", cr.Spec.IssuerRef.Name)
			_ = r.setStatus(ctx, cr, cmmeta.ConditionFalse, cmapi.CertificateRequestReasonPending, "StepClusterIssuer resource %s is not Ready", issNamespaceName)
			return ctrl.Result{}, err
		}
	
		// Load the provisioner that will sign the CertificateRequest
		provisioner, ok := provisioners.Load(issNamespaceName)
		if !ok {
			err := fmt.Errorf("provisioner %s not found", issNamespaceName)
			log.Error(err, "failed to provisioner for StepClusterIssuer resource")
			_ = r.setStatus(ctx, cr, cmmeta.ConditionFalse, cmapi.CertificateRequestReasonPending, "Failed to load provisioner for StepClusterIssuer resource %s", issNamespaceName)
			return ctrl.Result{}, err
		}
	
		// Sign CertificateRequest
		signedPEM, trustedCAs, err := provisioner.Sign(ctx, cr)
		if err != nil {
			log.Error(err, "failed to sign certificate request")
			return ctrl.Result{}, r.setStatus(ctx, cr, cmmeta.ConditionFalse, cmapi.CertificateRequestReasonFailed, "Failed to sign certificate request: %v", err)
		}
		cr.Status.Certificate = signedPEM
		cr.Status.CA = trustedCAs
	
		return ctrl.Result{}, r.setStatus(ctx, cr, cmmeta.ConditionTrue, cmapi.CertificateRequestReasonIssued, "Certificate issued")
	} else {
		iss := api.StepIssuer{}
		issNamespaceName := types.NamespacedName{
		  Namespace: req.Namespace,
		  Name:      cr.Spec.IssuerRef.Name,
	  }

		if err := r.Client.Get(ctx, issNamespaceName, &iss); err != nil {
			log.Error(err, "failed to retrieve StepIssuer resource", "namespace", req.Namespace, "name", cr.Spec.IssuerRef.Name)
			_ = r.setStatus(ctx, cr, cmmeta.ConditionFalse, cmapi.CertificateRequestReasonPending, "Failed to retrieve StepIssuer resource %s: %v", issNamespaceName, err)
			return ctrl.Result{}, err
		}

		// Check if the StepIssuer resource has been marked Ready
		if !stepIssuerHasCondition(iss, api.StepIssuerCondition{Type: api.ConditionReady, Status: api.ConditionTrue}) {
			err := fmt.Errorf("resource %s is not ready", issNamespaceName)
			log.Error(err, "failed to retrieve StepIssuer resource", "namespace", req.Namespace, "name", cr.Spec.IssuerRef.Name)
			_ = r.setStatus(ctx, cr, cmmeta.ConditionFalse, cmapi.CertificateRequestReasonPending, "StepIssuer resource %s is not Ready", issNamespaceName)
			return ctrl.Result{}, err
		}

		// Load the provisioner that will sign the CertificateRequest
		provisioner, ok := provisioners.Load(issNamespaceName)
		if !ok {
			err := fmt.Errorf("provisioner %s not found", issNamespaceName)
			log.Error(err, "failed to provisioner for StepIssuer resource")
			_ = r.setStatus(ctx, cr, cmmeta.ConditionFalse, cmapi.CertificateRequestReasonPending, "Failed to load provisioner for StepIssuer resource %s", issNamespaceName)
			return ctrl.Result{}, err
		}

		// Sign CertificateRequest
		signedPEM, trustedCAs, err := provisioner.Sign(ctx, cr)
		if err != nil {
			log.Error(err, "failed to sign certificate request")
			return ctrl.Result{}, r.setStatus(ctx, cr, cmmeta.ConditionFalse, cmapi.CertificateRequestReasonFailed, "Failed to sign certificate request: %v", err)
		}
		cr.Status.Certificate = signedPEM
		cr.Status.CA = trustedCAs

		return ctrl.Result{}, r.setStatus(ctx, cr, cmmeta.ConditionTrue, cmapi.CertificateRequestReasonIssued, "Certificate issued")
	}
}

// SetupWithManager initializes the CertificateRequest controller into the
// controller runtime.
func (r *CertificateRequestReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&cmapi.CertificateRequest{}).
		Complete(r)
}

// stepIssuerHasCondition will return true if the given StepIssuer resource has
// a condition matching the provided StepIssuerCondition. Only the Type and
// Status field will be used in the comparison, meaning that this function will
// return 'true' even if the Reason, Message and LastTransitionTime fields do
// not match.
func stepIssuerHasCondition(iss api.StepIssuer, c api.StepIssuerCondition) bool {
	existingConditions := iss.Status.Conditions
	for _, cond := range existingConditions {
		if c.Type == cond.Type && c.Status == cond.Status {
			return true
		}
	}
	return false
}

// stepClusterIssuerHasCondition will return true if the given StepClusterIssuer resource has
// a condition matching the provided StepClusterIssuerCondition. Only the Type and
// Status field will be used in the comparison, meaning that this function will
// return 'true' even if the Reason, Message and LastTransitionTime fields do
// not match.
func stepClusterIssuerHasCondition(iss api.StepClusterIssuer, c api.StepClusterIssuerCondition) bool {
	existingConditions := iss.Status.Conditions
	for _, cond := range existingConditions {
		if c.Type == cond.Type && c.Status == cond.Status {
			return true
		}
	}
	return false
}

func (r *CertificateRequestReconciler) setStatus(ctx context.Context, cr *cmapi.CertificateRequest, status cmmeta.ConditionStatus, reason, message string, args ...interface{}) error {
	completeMessage := fmt.Sprintf(message, args...)
	apiutil.SetCertificateRequestCondition(cr, cmapi.CertificateRequestConditionReady, status, reason, completeMessage)

	// Fire an Event to additionally inform users of the change
	eventType := core.EventTypeNormal
	if status == cmmeta.ConditionFalse {
		eventType = core.EventTypeWarning
	}
	r.Recorder.Event(cr, eventType, reason, completeMessage)

	return r.Client.Status().Update(ctx, cr)
}
