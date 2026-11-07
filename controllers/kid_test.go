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
	"testing"

	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestValidateProvisionerKeyIDSource(t *testing.T) {
	tests := []struct {
		name      string
		kid       string
		secret    bool
		configMap bool
		wantErr   bool
	}{
		{name: "literal ok", kid: "abc"},
		{name: "secret ok", secret: true},
		{name: "configmap ok", configMap: true},
		{name: "none set", wantErr: true},
		{name: "literal and secret", kid: "abc", secret: true, wantErr: true},
		{name: "secret and configmap", secret: true, configMap: true, wantErr: true},
		{name: "all three", kid: "abc", secret: true, configMap: true, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateProvisionerKeyIDSource(tt.kid, tt.secret, tt.configMap)
			if tt.wantErr != (err != nil) {
				t.Fatalf("wantErr=%v, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestResolveProvisionerKeyID(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := core.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
		&core.Secret{
			ObjectMeta: metav1.ObjectMeta{Namespace: "ns", Name: "s"},
			Data:       map[string][]byte{"kid": []byte("from-secret\n"), "empty": []byte(" ")},
		},
		&core.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Namespace: "ns", Name: "cm"},
			Data:       map[string]string{"kid": "from-configmap"},
		},
	).Build()
	ctx := context.Background()

	tests := []struct {
		name         string
		kid          string
		secret, cm   *keyRef
		want         string
		wantNotFound bool
		wantErr      bool
	}{
		{name: "literal", kid: "literal", want: "literal"},
		{name: "secret", secret: &keyRef{"ns", "s", "kid"}, want: "from-secret"},
		{name: "configmap", cm: &keyRef{"ns", "cm", "kid"}, want: "from-configmap"},
		{name: "secret missing", secret: &keyRef{"ns", "nope", "kid"}, wantErr: true, wantNotFound: true},
		{name: "secret key missing", secret: &keyRef{"ns", "s", "nope"}, wantErr: true, wantNotFound: true},
		{name: "secret empty value", secret: &keyRef{"ns", "s", "empty"}, wantErr: true},
		{name: "configmap missing", cm: &keyRef{"ns", "nope", "kid"}, wantErr: true, wantNotFound: true},
		{name: "configmap key missing", cm: &keyRef{"ns", "cm", "nope"}, wantErr: true, wantNotFound: true},
		{name: "no source", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, notFound, err := resolveProvisionerKeyID(ctx, c, tt.kid, tt.secret, tt.cm)
			if tt.wantErr != (err != nil) {
				t.Fatalf("wantErr=%v, got %v", tt.wantErr, err)
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
			if notFound != tt.wantNotFound {
				t.Errorf("notFound=%v, want %v", notFound, tt.wantNotFound)
			}
		})
	}
}
