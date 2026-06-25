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
	"bytes"
	"context"
	"fmt"
	"os"

	core "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// resolveProvisionerPassword loads the provisioner password from the first
// configured source: a Kubernetes Secret, an environment variable read from the
// controller's own environment, or a file on the controller's own filesystem.
// The environment variable and file sources let the password be injected into
// the controller pod (for example by Vault Agent) without storing it in a
// Kubernetes Secret.
//
// The returned bool reports whether a failure is a "not found" condition, so
// callers can set an accurate status reason.
func resolveProvisionerPassword(ctx context.Context, c client.Client, secretNamespace, secretName, secretKey, passwordEnv, passwordFile string) (password []byte, notFound bool, err error) {
	switch {
	case secretName != "":
		var secret core.Secret
		key := types.NamespacedName{Namespace: secretNamespace, Name: secretName}
		if err := c.Get(ctx, key, &secret); err != nil {
			return nil, apierrors.IsNotFound(err), fmt.Errorf("failed to retrieve provisioner secret: %w", err)
		}
		// The Secret source is intentionally returned verbatim to preserve
		// existing behavior.
		v, ok := secret.Data[secretKey]
		if !ok {
			return nil, true, fmt.Errorf("secret %s does not contain key %s", secret.Name, secretKey)
		}
		return v, false, nil
	case passwordEnv != "":
		v, ok := os.LookupEnv(passwordEnv)
		if !ok {
			return nil, true, fmt.Errorf("environment variable %s is not set", passwordEnv)
		}
		return trimPassword([]byte(v)), false, nil
	case passwordFile != "":
		v, err := os.ReadFile(passwordFile)
		if err != nil {
			return nil, os.IsNotExist(err), fmt.Errorf("failed to read provisioner password file %s: %w", passwordFile, err)
		}
		return trimPassword(v), false, nil
	default:
		// Should be unreachable: the spec is validated before reaching here.
		return nil, false, fmt.Errorf("no provisioner password source configured")
	}
}

// trimPassword removes trailing newline characters commonly appended by
// templating tools such as Vault Agent. It is applied to the environment
// variable and file sources only; Secret data is used verbatim.
func trimPassword(b []byte) []byte {
	return bytes.TrimRight(b, "\r\n")
}

// validateProvisionerPasswordSource ensures that exactly one provisioner
// password source is configured, and that a Secret reference includes a key.
func validateProvisionerPasswordSource(secretName, secretKey, passwordEnv, passwordFile string) error {
	sources := 0
	for _, s := range []string{secretName, passwordEnv, passwordFile} {
		if s != "" {
			sources++
		}
	}
	switch {
	case sources == 0:
		return fmt.Errorf("one of spec.provisioner.passwordRef, spec.provisioner.passwordEnv, or spec.provisioner.passwordFile must be set")
	case sources > 1:
		return fmt.Errorf("only one of spec.provisioner.passwordRef, spec.provisioner.passwordEnv, or spec.provisioner.passwordFile may be set")
	case secretName != "" && secretKey == "":
		return fmt.Errorf("spec.provisioner.passwordRef.key cannot be empty")
	default:
		return nil
	}
}
