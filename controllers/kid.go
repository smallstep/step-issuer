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
	"strings"

	api "github.com/smallstep/step-issuer/api/v1beta1"
	core "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// keyRef is a namespaced reference to a key in a Secret or ConfigMap.
type keyRef struct {
	Namespace string
	Name      string
	Key       string
}

// issuerKeyRef converts a StepIssuer key selector into a keyRef in the given
// namespace.
func issuerKeyRef(s *api.StepIssuerKeySelector, namespace string) *keyRef {
	if s == nil {
		return nil
	}
	return &keyRef{Namespace: namespace, Name: s.Name, Key: s.Key}
}

// clusterKeyRef converts a StepClusterIssuer key selector into a keyRef.
func clusterKeyRef(s *api.StepClusterIssuerKeySelector) *keyRef {
	if s == nil {
		return nil
	}
	return &keyRef{Namespace: s.Namespace, Name: s.Name, Key: s.Key}
}

// resolveProvisionerKeyID returns the provisioner kid from the configured
// source: the literal value, a key in a Secret, or a key in a ConfigMap.
//
// The returned bool reports whether a failure is a "not found" condition, so
// callers can set an accurate status reason.
func resolveProvisionerKeyID(ctx context.Context, c client.Client, kid string, secretRef, configMapRef *keyRef) (string, bool, error) {
	var value string
	switch {
	case kid != "":
		return kid, false, nil
	case secretRef != nil:
		var secret core.Secret
		key := types.NamespacedName{Namespace: secretRef.Namespace, Name: secretRef.Name}
		if err := c.Get(ctx, key, &secret); err != nil {
			return "", apierrors.IsNotFound(err), fmt.Errorf("failed to retrieve kid secret: %w", err)
		}
		v, ok := secret.Data[secretRef.Key]
		if !ok {
			return "", true, fmt.Errorf("secret %s does not contain key %s", secret.Name, secretRef.Key)
		}
		value = string(v)
	case configMapRef != nil:
		var cm core.ConfigMap
		key := types.NamespacedName{Namespace: configMapRef.Namespace, Name: configMapRef.Name}
		if err := c.Get(ctx, key, &cm); err != nil {
			return "", apierrors.IsNotFound(err), fmt.Errorf("failed to retrieve kid configmap: %w", err)
		}
		v, ok := cm.Data[configMapRef.Key]
		if !ok {
			return "", true, fmt.Errorf("configmap %s does not contain key %s", cm.Name, configMapRef.Key)
		}
		value = v
	default:
		// Should be unreachable: the spec is validated before reaching here.
		return "", false, fmt.Errorf("no provisioner kid source configured")
	}

	value = strings.TrimSpace(value)
	if value == "" {
		return "", false, fmt.Errorf("provisioner kid referenced from the cluster is empty")
	}
	return value, false, nil
}

// validateProvisionerKeyIDSource ensures that exactly one provisioner kid
// source is configured.
func validateProvisionerKeyIDSource(kid string, hasSecretRef, hasConfigMapRef bool) error {
	sources := 0
	for _, set := range []bool{kid != "", hasSecretRef, hasConfigMapRef} {
		if set {
			sources++
		}
	}
	switch {
	case sources == 0:
		return fmt.Errorf("one of spec.provisioner.kid, spec.provisioner.kidSecretRef, or spec.provisioner.kidConfigMapRef must be set")
	case sources > 1:
		return fmt.Errorf("only one of spec.provisioner.kid, spec.provisioner.kidSecretRef, or spec.provisioner.kidConfigMapRef may be set")
	default:
		return nil
	}
}
