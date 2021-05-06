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
	SchemeBuilder.Register(&StepClusterIssuer{}, &StepClusterIssuerList{})
}

// StepClusterIssuerSpec defines the desired state of StepClusterIssuer
type StepClusterIssuerSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// URL is the base URL for the step certificates instance.
	URL string `json:"url"`

	// Provisioner contains the step certificates provisioner configuration.
	Provisioner StepClusterProvisioner `json:"provisioner"`

	// CABundle is a base64 encoded TLS certificate used to verify connections
	// to the step certificates server. If not set the system root certificates
	// are used to validate the TLS connection.
	// +optional
	CABundle []byte `json:"caBundle,omitempty"`
}

// StepClusterIssuerStatus defines the observed state of StepClusterIssuer
type StepClusterIssuerStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// +optional
	Conditions []StepClusterIssuerCondition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster

// StepClusterIssuer is the Schema for the stepclusterissuers API
// +kubebuilder:subresource:status
type StepClusterIssuer struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   StepClusterIssuerSpec   `json:"spec,omitempty"`
	Status StepClusterIssuerStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// StepClusterIssuerList contains a list of StepClusterIssuer
type StepClusterIssuerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []StepClusterIssuer `json:"items"`
}

// StepClusterIssuerSecretKeySelector contains the reference to a secret.
type StepClusterIssuerSecretKeySelector struct {
	// The name of the secret in the pod's namespace to select from.
	Name string `json:"name"`

	// The namespace of the secret in the pod's namespace to select from.
	Namespace string `json:"namespace"`

	// The key of the secret to select from. Must be a valid secret key.
	// +optional
	Key string `json:"key,omitempty"`
}

// StepClusterProvisioner contains the configuration used to create step certificate
// tokens used to grant certificates.
type StepClusterProvisioner struct {
	// Names is the name of the JWK provisioner.
	Name string `json:"name"`

	// KeyID is the kid property of the JWK provisioner.
	KeyID string `json:"kid"`

	// PasswordRef is a reference to a Secret containing the provisioner
	// password used to decrypt the provisioner private key.
	PasswordRef StepClusterIssuerSecretKeySelector `json:"passwordRef"`
}

// StepClusterIssuerCondition contains condition information for the step issuer.
type StepClusterIssuerCondition struct {
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
