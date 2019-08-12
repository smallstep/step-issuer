package controllers

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	api "github.com/smallstep/step-issuer/api/v1beta1"
	core "k8s.io/api/core/v1"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type stepStatusReconciler struct {
	*StepIssuerReconciler
	issuer *api.StepIssuer
	logger logr.Logger
}

func newStepStatusReconciler(r *StepIssuerReconciler, iss *api.StepIssuer, log logr.Logger) *stepStatusReconciler {
	return &stepStatusReconciler{
		StepIssuerReconciler: r,
		issuer:               iss,
		logger:               log,
	}
}

func (r *stepStatusReconciler) Update(ctx context.Context, status api.ConditionStatus, reason, message string, args ...interface{}) error {
	completeMessage := fmt.Sprintf(message, args...)
	r.setCondition(status, reason, completeMessage)

	// Fire an Event to additionally inform users of the change
	eventType := core.EventTypeNormal
	if status == api.ConditionFalse {
		eventType = core.EventTypeWarning
	}
	r.Recorder.Event(r.issuer, eventType, reason, completeMessage)

	return r.Client.Update(ctx, r.issuer)
}

// setCondition will set a 'condition' on the given api.StepIssuer resource.
//
// - If no condition of the same type already exists, the condition will be
//   inserted with the LastTransitionTime set to the current time.
// - If a condition of the same type and state already exists, the condition
//   will be updated but the LastTransitionTime will not be modified.
// - If a condition of the same type and different state already exists, the
//   condition will be updated and the LastTransitionTime set to the current
//   time.
func (r *stepStatusReconciler) setCondition(status api.ConditionStatus, reason, message string) {
	now := meta.NewTime(r.Clock.Now())
	c := api.StepIssuerCondition{
		Type:               api.ConditionReady,
		Status:             status,
		Reason:             reason,
		Message:            message,
		LastTransitionTime: &now,
	}

	// Search through existing conditions
	for idx, cond := range r.issuer.Status.Conditions {
		// Skip unrelated conditions
		if cond.Type != api.ConditionReady {
			continue
		}

		// If this update doesn't contain a state transition, we don't update
		// the conditions LastTransitionTime to Now()
		if cond.Status == status {
			c.LastTransitionTime = cond.LastTransitionTime
		} else {
			r.logger.Info("found status change for StepIssuer condition; setting lastTransitionTime", "condition", cond.Type, "old_status", cond.Status, "new_status", status, "time", now.Time)
		}

		// Overwrite the existing condition
		r.issuer.Status.Conditions[idx] = c
		return
	}

	// If we've not found an existing condition of this type, we simply insert
	// the new condition into the slice.
	r.issuer.Status.Conditions = append(r.issuer.Status.Conditions, c)
	r.logger.Info("setting lastTransitionTime for StepIssuer condition", "condition", api.ConditionReady, "time", now.Time)
}
