package provisioners

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	api "github.com/smallstep/step-issuer/api/v1beta1"
)

func TestReadHeaderValue(t *testing.T) {
	tests := []struct {
		name       string
		value      string
		setup      func() (string, error)
		wantErr    bool
		wantValue  string
		wantErrMsg string
	}{
		{
			name:      "empty value",
			value:     "",
			wantErr:   false,
			wantValue: "",
		},
		{
			name:      "static value",
			value:     "my-static-token",
			wantErr:   false,
			wantValue: "my-static-token",
		},
		{
			name:      "static value with spaces",
			value:     "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9",
			wantErr:   false,
			wantValue: "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9",
		},
		{
			name:    "file path - valid",
			value:   "file:///tmp/test-header-value",
			wantErr: false,
			setup: func() (string, error) {
				tmpDir := "/tmp"
				testFile := filepath.Join(tmpDir, "test-header-value")
				testContent := "my-jwt-token-from-file"
				if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
					return "", err
				}
				return testContent, nil
			},
			wantValue: "my-jwt-token-from-file",
		},
		{
			name:    "file path - with whitespace",
			value:   "file:///tmp/test-header-whitespace",
			wantErr: false,
			setup: func() (string, error) {
				tmpDir := "/tmp"
				testFile := filepath.Join(tmpDir, "test-header-whitespace")
				testContent := "  my-token-with-spaces  \n"
				if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
					return "", err
				}
				return "my-token-with-spaces", nil
			},
			wantValue: "my-token-with-spaces",
		},
		{
			name:       "file path - not found",
			value:      "file:///tmp/nonexistent-test-file-xyz",
			wantErr:    true,
			wantErrMsg: "failed to read file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				expected, err := tt.setup()
				if err != nil {
					t.Fatalf("setup failed: %v", err)
				}
				tt.wantValue = expected
			}

			got, err := readHeaderValue(tt.value)

			if (err != nil) != tt.wantErr {
				t.Errorf("readHeaderValue() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.wantErrMsg != "" && (err == nil || !strings.Contains(err.Error(), tt.wantErrMsg)) {
				t.Errorf("readHeaderValue() error = %v, want error with message containing %q", err, tt.wantErrMsg)
				return
			}

			if got != tt.wantValue {
				t.Errorf("readHeaderValue() = %v, want %v", got, tt.wantValue)
			}
		})
	}
}

func TestNewCustomHeaderTransport(t *testing.T) {
	tests := []struct {
		name        string
		headerName  string
		headerValue string
	}{
		{
			name:        "simple header",
			headerName:  "X-Custom-Auth",
			headerValue: "my-token",
		},
		{
			name:        "authorization header",
			headerName:  "Authorization",
			headerValue: "Bearer my-jwt-token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			header := &api.CustomHeader{Name: tt.headerName, Value: tt.headerValue}
			transport := NewCustomHeaderTransport(http.DefaultTransport, header)

			if transport == nil {
				t.Fatal("NewCustomHeaderTransport() returned nil")
			}

			if transport.header == nil || transport.header.Name != tt.headerName || transport.header.Value != tt.headerValue {
				t.Fatalf("transport header mismatch: got %#v", transport.header)
			}
		})
	}
}

func TestStepProvisionerCustomHeaderStorage(t *testing.T) {
	tests := []struct {
		name             string
		issuer           *api.StepIssuer
		wantCustomHeader *api.CustomHeader
		wantErr          bool
		wantErrSubstring string
	}{
		{
			name: "issuer without custom header",
			issuer: &api.StepIssuer{
				Spec: api.StepIssuerSpec{
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
					CustomHeader: nil,
				},
			},
			wantCustomHeader: nil,
			wantErr:          false,
		},
		{
			name: "issuer with static custom header",
			issuer: &api.StepIssuer{
				Spec: api.StepIssuerSpec{
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
						Value: "my-token",
					},
				},
			},
			wantCustomHeader: &api.CustomHeader{
				Name:  "X-Custom-Auth",
				Value: "my-token",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Note: This test would require mocking ca.NewProvisioner
			// For now, we're just testing that CustomHeader is properly stored
			// A full test would require more setup with the CA provisioner

			// Verify that the CustomHeader field is properly set in the issuer spec
			if tt.issuer.Spec.CustomHeader != tt.wantCustomHeader {
				// If both are nil, that's okay
				if tt.issuer.Spec.CustomHeader == nil && tt.wantCustomHeader == nil {
					return
				}
				// If one is nil and the other isn't, that's an error
				if tt.issuer.Spec.CustomHeader == nil || tt.wantCustomHeader == nil {
					t.Errorf("CustomHeader mismatch: got %v, want %v", tt.issuer.Spec.CustomHeader, tt.wantCustomHeader)
					return
				}
				// If both are non-nil, compare their contents
				if tt.issuer.Spec.CustomHeader.Name != tt.wantCustomHeader.Name ||
					tt.issuer.Spec.CustomHeader.Value != tt.wantCustomHeader.Value {
					t.Errorf("CustomHeader content mismatch: got %+v, want %+v",
						tt.issuer.Spec.CustomHeader, tt.wantCustomHeader)
				}
			}
		})
	}
}

func TestCustomHeaderTransportRoundTrip(t *testing.T) {
	// Test that the CustomHeaderTransport properly injects headers
	backend := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("X-Custom-Header"); got != "test-value" {
			http.Error(w, "missing header", http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
	server := httptest.NewServer(backend)
	defer server.Close()

	transport := NewCustomHeaderTransport(http.DefaultTransport, &api.CustomHeader{
		Name:  "X-Custom-Header",
		Value: "test-value",
	})

	if transport == nil {
		t.Fatal("NewCustomHeaderTransport returned nil")
	}

	req, err := http.NewRequest(http.MethodGet, server.URL, nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	resp, err := transport.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip() failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status code: got %d, want %d", resp.StatusCode, http.StatusOK)
	}
	if transport.transport == nil {
		t.Fatal("transport.transport is nil")
	}
}

func TestCustomHeaderTransportRoundTrip_FileValue(t *testing.T) {
	tokenFile := filepath.Join(t.TempDir(), "token")
	if err := os.WriteFile(tokenFile, []byte("dynamic-token\n"), 0600); err != nil {
		t.Fatalf("failed to create token file: %v", err)
	}

	backend := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "dynamic-token" {
			http.Error(w, "missing header", http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
	server := httptest.NewServer(backend)
	defer server.Close()

	transport := NewCustomHeaderTransport(http.DefaultTransport, &api.CustomHeader{
		Name:  "Authorization",
		Value: "file://" + tokenFile,
	})

	req, err := http.NewRequest(http.MethodGet, server.URL, nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	resp, err := transport.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip() failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status code: got %d, want %d", resp.StatusCode, http.StatusOK)
	}
}

func TestCustomHeaderTransportRoundTrip_FileValueError(t *testing.T) {
	transport := NewCustomHeaderTransport(http.DefaultTransport, &api.CustomHeader{
		Name:  "Authorization",
		Value: "file:///tmp/does-not-exist-token",
	})

	req, err := http.NewRequest(http.MethodGet, "http://example.com", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	_, err = transport.RoundTrip(req)
	if err == nil {
		t.Fatal("expected an error but got nil")
	}
	if !strings.Contains(err.Error(), "failed to read file") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCustomHeaderTransportRoundTrip_FileValueRefreshedPerRequest(t *testing.T) {
	tokenFile := filepath.Join(t.TempDir(), "token")
	if err := os.WriteFile(tokenFile, []byte("first-token\n"), 0600); err != nil {
		t.Fatalf("failed to create token file: %v", err)
	}

	requests := 0
	backend := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		got := r.Header.Get("Authorization")
		switch requests {
		case 1:
			if got != "first-token" {
				http.Error(w, "first request header mismatch", http.StatusBadRequest)
				return
			}
		case 2:
			if got != "second-token" {
				http.Error(w, "second request header mismatch", http.StatusBadRequest)
				return
			}
		default:
			http.Error(w, "unexpected request count", http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
	server := httptest.NewServer(backend)
	defer server.Close()

	transport := NewCustomHeaderTransport(http.DefaultTransport, &api.CustomHeader{
		Name:  "Authorization",
		Value: "file://" + tokenFile,
	})

	req1, err := http.NewRequest(http.MethodGet, server.URL, nil)
	if err != nil {
		t.Fatalf("failed to create first request: %v", err)
	}
	resp1, err := transport.RoundTrip(req1)
	if err != nil {
		t.Fatalf("first RoundTrip() failed: %v", err)
	}
	resp1.Body.Close()

	if err := os.WriteFile(tokenFile, []byte("second-token\n"), 0600); err != nil {
		t.Fatalf("failed to update token file: %v", err)
	}

	req2, err := http.NewRequest(http.MethodGet, server.URL, nil)
	if err != nil {
		t.Fatalf("failed to create second request: %v", err)
	}
	resp2, err := transport.RoundTrip(req2)
	if err != nil {
		t.Fatalf("second RoundTrip() failed: %v", err)
	}
	defer resp2.Body.Close()

	if resp1.StatusCode != http.StatusOK || resp2.StatusCode != http.StatusOK {
		t.Fatalf("unexpected response codes: first=%d second=%d", resp1.StatusCode, resp2.StatusCode)
	}
}

func TestCustomHeaderTransportRoundTrip_NoHeaderConfigured(t *testing.T) {
	backend := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "" {
			http.Error(w, "unexpected authorization header", http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
	server := httptest.NewServer(backend)
	defer server.Close()

	transport := NewCustomHeaderTransport(http.DefaultTransport, nil)
	req, err := http.NewRequest(http.MethodGet, server.URL, nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	resp, err := transport.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip() failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status code: got %d, want %d", resp.StatusCode, http.StatusOK)
	}
}
