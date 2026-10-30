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
	"testing"

	api "github.com/smallstep/step-issuer/api/v1beta1"
)

func validStepIssuerSpec() api.StepIssuerSpec {
	return api.StepIssuerSpec{
		URL: "https://ca.example.com",
		Provisioner: api.StepProvisioner{
			Name:        "admin",
			KeyID:       "kid",
			PasswordRef: api.StepIssuerSecretKeySelector{Name: "s", Key: "password"},
		},
	}
}

func validStepClusterIssuerSpec() api.StepClusterIssuerSpec {
	return api.StepClusterIssuerSpec{
		URL: "https://ca.example.com",
		Provisioner: api.StepClusterProvisioner{
			Name:        "admin",
			KeyID:       "kid",
			PasswordRef: api.StepClusterIssuerSecretKeySelector{Name: "s", Namespace: "ns", Key: "password"},
		},
	}
}

func TestValidateStepIssuerSpecCABundleIsOptional(t *testing.T) {
	// A spec without a CABundle must be considered valid: the provisioner is
	// expected to fall back to the system trust store.
	spec := validStepIssuerSpec()
	spec.CABundle = nil
	if err := validateStepIssuerSpec(spec); err != nil {
		t.Fatalf("expected no error for empty caBundle, got %v", err)
	}

	// A spec with a CABundle set must remain valid too.
	spec.CABundle = []byte("-----BEGIN CERTIFICATE-----\n...\n-----END CERTIFICATE-----\n")
	if err := validateStepIssuerSpec(spec); err != nil {
		t.Fatalf("expected no error for a non-empty caBundle, got %v", err)
	}
}

func TestValidateStepIssuerSpecStillValidatesOtherFields(t *testing.T) {
	spec := validStepIssuerSpec()
	spec.URL = ""
	if err := validateStepIssuerSpec(spec); err == nil {
		t.Fatal("expected an error when spec.url is empty")
	}
}

func TestValidateStepClusterIssuerSpecCABundleIsOptional(t *testing.T) {
	spec := validStepClusterIssuerSpec()
	spec.CABundle = nil
	if err := validateStepClusterIssuerSpec(spec); err != nil {
		t.Fatalf("expected no error for empty caBundle, got %v", err)
	}

	spec.CABundle = []byte("-----BEGIN CERTIFICATE-----\n...\n-----END CERTIFICATE-----\n")
	if err := validateStepClusterIssuerSpec(spec); err != nil {
		t.Fatalf("expected no error for a non-empty caBundle, got %v", err)
	}
}

func TestValidateStepClusterIssuerSpecStillValidatesOtherFields(t *testing.T) {
	spec := validStepClusterIssuerSpec()
	spec.URL = ""
	if err := validateStepClusterIssuerSpec(spec); err == nil {
		t.Fatal("expected an error when spec.url is empty")
	}
}
