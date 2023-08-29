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

package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

func init() {
	SchemeBuilder.Register(&StepIssuer{}, &StepIssuerList{})
}

// StepIssuerSpec defines the desired state of StepIssuer
type StepIssuerSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// URL is the base URL for the step certificates instance.
	URL string `json:"url"`

	// Provisioner contains the step certificates provisioner configuration.
	Provisioner StepProvisioner `json:"provisioner"`

	// CABundle is a base64 encoded TLS certificate used to verify connections
	// to the step certificates server. If not set the system root certificates
	// are used to validate the TLS connection.
	CABundle []byte `json:"caBundle"`
}

// StepIssuerStatus defines the observed state of StepIssuer
type StepIssuerStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// +optional
	Conditions []StepIssuerCondition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true

// StepIssuer is the Schema for the stepissuers API
// +kubebuilder:subresource:status
type StepIssuer struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   StepIssuerSpec   `json:"spec,omitempty"`
	Status StepIssuerStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// StepIssuerList contains a list of StepIssuer
type StepIssuerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []StepIssuer `json:"items"`
}

// StepIssuerSecretKeySelector contains the reference to a secret.
type StepIssuerSecretKeySelector struct {
	// The name of the secret in the pod's namespace to select from.
	Name string `json:"name"`

	// The key of the secret to select from. Must be a valid secret key.
	// +optional
	Key string `json:"key,omitempty"`
}

// StepProvisioner contains the configuration used to create step certificate
// tokens used to grant certificates.
type StepProvisioner struct {
	// Names is the name of the JWK provisioner.
	Name string `json:"name"`

	// KeyID is the kid property of the JWK provisioner.
	KeyID string `json:"kid"`

	// PasswordRef is a reference to a Secret containing the provisioner
	// password used to decrypt the provisioner private key.
	PasswordRef StepIssuerSecretKeySelector `json:"passwordRef"`
}

// ConditionType represents a StepIssuer condition type.
// +kubebuilder:validation:Enum=Ready
type ConditionType string

const (
	// ConditionReady indicates that a StepIssuer is ready for use.
	ConditionReady ConditionType = "Ready"
)

// ConditionStatus represents a condition's status.
// +kubebuilder:validation:Enum=True;False;Unknown
type ConditionStatus string

// These are valid condition statuses. "ConditionTrue" means a resource is in
// the condition; "ConditionFalse" means a resource is not in the condition;
// "ConditionUnknown" means kubernetes can't decide if a resource is in the
// condition or not. In the future, we could add other intermediate
// conditions, e.g. ConditionDegraded.
const (
	// ConditionTrue represents the fact that a given condition is true
	ConditionTrue ConditionStatus = "True"

	// ConditionFalse represents the fact that a given condition is false
	ConditionFalse ConditionStatus = "False"

	// ConditionUnknown represents the fact that a given condition is unknown
	ConditionUnknown ConditionStatus = "Unknown"
)

// StepIssuerCondition contains condition information for the step issuer.
type StepIssuerCondition struct {
	// Type of the condition, currently ('Ready').
	Type ConditionType `json:"type"`

	// Status of the condition, one of ('True', 'False', 'Unknown').
	// +kubebuilder:validation:Enum=True;False;Unknown
	Status ConditionStatus `json:"status"`

	// LastTransitionTime is the timestamp corresponding to the last status
	// change of this condition.
	// +optional
	LastTransitionTime *metav1.Time `json:"lastTransitionTime,omitempty"`

	// Reason is a brief machine readable explanation for the condition's last
	// transition.
	// +optional
	Reason string `json:"reason,omitempty"`

	// Message is a human readable description of the details of the last
	// transition, complementing reason.
	// +optional
	Message string `json:"message,omitempty"`
}
