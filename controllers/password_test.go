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
	"os"
	"path/filepath"
	"testing"
)

func TestValidateProvisionerPasswordSource(t *testing.T) {
	tests := []struct {
		name         string
		secretName   string
		secretKey    string
		passwordEnv  string
		passwordFile string
		wantErr      bool
	}{
		{name: "secret ok", secretName: "s", secretKey: "password"},
		{name: "env ok", passwordEnv: "STEP_PASSWORD"},
		{name: "file ok", passwordFile: "/etc/step/password"},
		{name: "none set", wantErr: true},
		{name: "secret without key", secretName: "s", wantErr: true},
		{name: "secret and env", secretName: "s", secretKey: "password", passwordEnv: "STEP_PASSWORD", wantErr: true},
		{name: "env and file", passwordEnv: "STEP_PASSWORD", passwordFile: "/etc/step/password", wantErr: true},
		{name: "all three", secretName: "s", secretKey: "password", passwordEnv: "E", passwordFile: "/f", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateProvisionerPasswordSource(tt.secretName, tt.secretKey, tt.passwordEnv, tt.passwordFile)
			if tt.wantErr && err == nil {
				t.Fatal("expected an error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
		})
	}
}

func TestResolveProvisionerPasswordEnv(t *testing.T) {
	t.Setenv("STEP_TEST_PASSWORD", "s3cr3t\n")
	got, notFound, err := resolveProvisionerPassword(context.Background(), nil, "", "", "", "STEP_TEST_PASSWORD", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if notFound {
		t.Fatal("unexpected notFound=true")
	}
	if string(got) != "s3cr3t" {
		t.Fatalf("expected trailing newline to be trimmed, got %q", string(got))
	}
}

func TestResolveProvisionerPasswordEnvUnset(t *testing.T) {
	_, notFound, err := resolveProvisionerPassword(context.Background(), nil, "", "", "", "STEP_TEST_PASSWORD_UNSET", "")
	if err == nil {
		t.Fatal("expected an error for an unset environment variable")
	}
	if !notFound {
		t.Fatal("expected notFound=true for an unset environment variable")
	}
}

func TestResolveProvisionerPasswordFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "password")
	if err := os.WriteFile(path, []byte("file-secret\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	got, notFound, err := resolveProvisionerPassword(context.Background(), nil, "", "", "", "", path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if notFound {
		t.Fatal("unexpected notFound=true")
	}
	if string(got) != "file-secret" {
		t.Fatalf("expected trailing newline to be trimmed, got %q", string(got))
	}
}

func TestResolveProvisionerPasswordFileMissing(t *testing.T) {
	_, notFound, err := resolveProvisionerPassword(context.Background(), nil, "", "", "", "", filepath.Join(t.TempDir(), "does-not-exist"))
	if err == nil {
		t.Fatal("expected an error for a missing file")
	}
	if !notFound {
		t.Fatal("expected notFound=true for a missing file")
	}
}

func TestResolveProvisionerPasswordNoSource(t *testing.T) {
	if _, _, err := resolveProvisionerPassword(context.Background(), nil, "", "", "", "", ""); err == nil {
		t.Fatal("expected an error when no password source is configured")
	}
}
