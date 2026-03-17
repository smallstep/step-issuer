package controllers

import (
	"testing"

	api "github.com/smallstep/step-issuer/api/v1beta1"
)

func TestValidateStepIssuerSpec(t *testing.T) {
	tests := []struct {
		name       string
		spec       api.StepIssuerSpec
		wantErr    bool
		wantErrMsg string
	}{
		{
			name: "valid spec without custom header",
			spec: api.StepIssuerSpec{
				URL:      "https://step-ca.local",
				CABundle: []byte("test-ca-bundle"),
				Provisioner: api.StepProvisioner{
					Name:  "admin",
					KeyID: "key-1",
					PasswordRef: api.StepIssuerSecretKeySelector{
						Name: "secret",
						Key:  "password",
					},
				},
				CustomHeader: nil,
			},
			wantErr: false,
		},
		{
			name: "valid spec with static custom header",
			spec: api.StepIssuerSpec{
				URL:      "https://step-ca.local",
				CABundle: []byte("test-ca-bundle"),
				Provisioner: api.StepProvisioner{
					Name:  "admin",
					KeyID: "key-1",
					PasswordRef: api.StepIssuerSecretKeySelector{
						Name: "secret",
						Key:  "password",
					},
				},
				CustomHeader: &api.CustomHeader{
					Name:  "X-Custom-Auth",
					Value: "my-token",
				},
			},
			wantErr: false,
		},
		{
			name: "valid spec with file-based custom header",
			spec: api.StepIssuerSpec{
				URL:      "https://step-ca.local",
				CABundle: []byte("test-ca-bundle"),
				Provisioner: api.StepProvisioner{
					Name:  "admin",
					KeyID: "key-1",
					PasswordRef: api.StepIssuerSecretKeySelector{
						Name: "secret",
						Key:  "password",
					},
				},
				CustomHeader: &api.CustomHeader{
					Name:  "Authorization",
					Value: "file:///etc/step-issuer/jwt",
				},
			},
			wantErr: false,
		},
		{
			name: "invalid spec - empty URL",
			spec: api.StepIssuerSpec{
				URL:      "",
				CABundle: []byte("test"),
				Provisioner: api.StepProvisioner{
					Name:  "admin",
					KeyID: "key-1",
					PasswordRef: api.StepIssuerSecretKeySelector{
						Name: "secret",
						Key:  "password",
					},
				},
			},
			wantErr:    true,
			wantErrMsg: "spec.url cannot be empty",
		},
		{
			name: "invalid spec - empty CABundle",
			spec: api.StepIssuerSpec{
				URL:      "https://step-ca.local",
				CABundle: []byte{},
				Provisioner: api.StepProvisioner{
					Name:  "admin",
					KeyID: "key-1",
					PasswordRef: api.StepIssuerSecretKeySelector{
						Name: "secret",
						Key:  "password",
					},
				},
			},
			wantErr:    true,
			wantErrMsg: "spec.caBundle cannot be empty",
		},
		{
			name: "invalid spec - empty provisioner name",
			spec: api.StepIssuerSpec{
				URL:      "https://step-ca.local",
				CABundle: []byte("test"),
				Provisioner: api.StepProvisioner{
					Name:  "",
					KeyID: "key-1",
					PasswordRef: api.StepIssuerSecretKeySelector{
						Name: "secret",
						Key:  "password",
					},
				},
			},
			wantErr:    true,
			wantErrMsg: "spec.provisioner.name cannot be empty",
		},
		{
			name: "invalid spec - empty provisioner KeyID",
			spec: api.StepIssuerSpec{
				URL:      "https://step-ca.local",
				CABundle: []byte("test"),
				Provisioner: api.StepProvisioner{
					Name:  "admin",
					KeyID: "",
					PasswordRef: api.StepIssuerSecretKeySelector{
						Name: "secret",
						Key:  "password",
					},
				},
			},
			wantErr:    true,
			wantErrMsg: "spec.provisioner.kid cannot be empty",
		},
		{
			name: "invalid spec - empty password ref name",
			spec: api.StepIssuerSpec{
				URL:      "https://step-ca.local",
				CABundle: []byte("test"),
				Provisioner: api.StepProvisioner{
					Name:  "admin",
					KeyID: "key-1",
					PasswordRef: api.StepIssuerSecretKeySelector{
						Name: "",
						Key:  "password",
					},
				},
			},
			wantErr:    true,
			wantErrMsg: "spec.provisioner.passwordRef.name cannot be empty",
		},
		{
			name: "invalid spec - empty password ref key",
			spec: api.StepIssuerSpec{
				URL:      "https://step-ca.local",
				CABundle: []byte("test"),
				Provisioner: api.StepProvisioner{
					Name:  "admin",
					KeyID: "key-1",
					PasswordRef: api.StepIssuerSecretKeySelector{
						Name: "secret",
						Key:  "",
					},
				},
			},
			wantErr:    true,
			wantErrMsg: "spec.provisioner.passwordRef.key cannot be empty",
		},
		{
			name: "invalid spec - custom header with empty name",
			spec: api.StepIssuerSpec{
				URL:      "https://step-ca.local",
				CABundle: []byte("test"),
				Provisioner: api.StepProvisioner{
					Name:  "admin",
					KeyID: "key-1",
					PasswordRef: api.StepIssuerSecretKeySelector{
						Name: "secret",
						Key:  "password",
					},
				},
				CustomHeader: &api.CustomHeader{
					Name:  "",
					Value: "my-token",
				},
			},
			wantErr:    true,
			wantErrMsg: "spec.customHeader.name cannot be empty",
		},
		{
			name: "invalid spec - custom header with empty value",
			spec: api.StepIssuerSpec{
				URL:      "https://step-ca.local",
				CABundle: []byte("test"),
				Provisioner: api.StepProvisioner{
					Name:  "admin",
					KeyID: "key-1",
					PasswordRef: api.StepIssuerSecretKeySelector{
						Name: "secret",
						Key:  "password",
					},
				},
				CustomHeader: &api.CustomHeader{
					Name:  "X-Custom-Auth",
					Value: "",
				},
			},
			wantErr:    true,
			wantErrMsg: "spec.customHeader.value cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateStepIssuerSpec(tt.spec)

			if (err != nil) != tt.wantErr {
				t.Errorf("validateStepIssuerSpec() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.wantErrMsg != "" {
				if err == nil || err.Error() != tt.wantErrMsg {
					t.Errorf("validateStepIssuerSpec() error message = %q, want %q", err, tt.wantErrMsg)
				}
			}
		})
	}
}

func TestValidateStepClusterIssuerSpec(t *testing.T) {
	tests := []struct {
		name       string
		spec       api.StepClusterIssuerSpec
		wantErr    bool
		wantErrMsg string
	}{
		{
			name: "valid spec without custom header",
			spec: api.StepClusterIssuerSpec{
				URL:      "https://step-ca.local",
				CABundle: []byte("test-ca-bundle"),
				Provisioner: api.StepClusterProvisioner{
					Name:  "admin",
					KeyID: "key-1",
					PasswordRef: api.StepClusterIssuerSecretKeySelector{
						Name:      "secret",
						Namespace: "default",
						Key:       "password",
					},
				},
				CustomHeader: nil,
			},
			wantErr: false,
		},
		{
			name: "valid spec with custom header",
			spec: api.StepClusterIssuerSpec{
				URL:      "https://step-ca.local",
				CABundle: []byte("test-ca-bundle"),
				Provisioner: api.StepClusterProvisioner{
					Name:  "admin",
					KeyID: "key-1",
					PasswordRef: api.StepClusterIssuerSecretKeySelector{
						Name:      "secret",
						Namespace: "default",
						Key:       "password",
					},
				},
				CustomHeader: &api.CustomHeader{
					Name:  "Authorization",
					Value: "file:///etc/step-issuer/jwt",
				},
			},
			wantErr: false,
		},
		{
			name: "invalid spec - custom header with empty name",
			spec: api.StepClusterIssuerSpec{
				URL:      "https://step-ca.local",
				CABundle: []byte("test"),
				Provisioner: api.StepClusterProvisioner{
					Name:  "admin",
					KeyID: "key-1",
					PasswordRef: api.StepClusterIssuerSecretKeySelector{
						Name:      "secret",
						Namespace: "default",
						Key:       "password",
					},
				},
				CustomHeader: &api.CustomHeader{
					Name:  "",
					Value: "my-token",
				},
			},
			wantErr:    true,
			wantErrMsg: "spec.customHeader.name cannot be empty",
		},
		{
			name: "invalid spec - custom header with empty value",
			spec: api.StepClusterIssuerSpec{
				URL:      "https://step-ca.local",
				CABundle: []byte("test"),
				Provisioner: api.StepClusterProvisioner{
					Name:  "admin",
					KeyID: "key-1",
					PasswordRef: api.StepClusterIssuerSecretKeySelector{
						Name:      "secret",
						Namespace: "default",
						Key:       "password",
					},
				},
				CustomHeader: &api.CustomHeader{
					Name:  "X-Header",
					Value: "",
				},
			},
			wantErr:    true,
			wantErrMsg: "spec.customHeader.value cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateStepClusterIssuerSpec(tt.spec)

			if (err != nil) != tt.wantErr {
				t.Errorf("validateStepClusterIssuerSpec() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.wantErrMsg != "" {
				if err == nil || err.Error() != tt.wantErrMsg {
					t.Errorf("validateStepClusterIssuerSpec() error message = %q, want %q", err, tt.wantErrMsg)
				}
			}
		})
	}
}
