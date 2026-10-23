package provisioners

import (
	"net/http"
	"testing"

	"github.com/smallstep/certificates/ca"
)

func TestSystemTrustStoreTransportLeavesRootCAsUnset(t *testing.T) {
	tr, ok := systemTrustStoreTransport().(*http.Transport)
	if !ok {
		t.Fatalf("expected *http.Transport, got %T", systemTrustStoreTransport())
	}
	if tr.TLSClientConfig == nil {
		t.Fatal("expected TLSClientConfig to be set")
	}
	// RootCAs must stay nil so that Go's crypto/tls falls back to verifying
	// against the host's/container's system trust store.
	if tr.TLSClientConfig.RootCAs != nil {
		t.Fatal("expected RootCAs to be nil so the system trust store is used")
	}
}

func TestCAClientOptionEmptyBundleFallsBackToSystemTrustStore(t *testing.T) {
	// Previously, an empty/unset caBundle was rejected by validation before
	// ever reaching the CA client. Now it must build a working client that
	// relies on the system trust store instead of erroring out.
	if _, err := ca.NewClient("https://ca.example.com", caClientOption(nil)); err != nil {
		t.Fatalf("expected no error configuring the client with an empty caBundle, got %v", err)
	}
}

func TestCAClientOptionNonEmptyBundleUsesCABundle(t *testing.T) {
	bundle := []byte("-----BEGIN CERTIFICATE-----\n" +
		"MIIBizCCATGgAwIBAgIQO+EAh8y/0V9P0XpHrVj5NTAKBggqhkjOPQQDAjAkMSIw\n" +
		"IAYDVQQDExlTdGVwIENlcnRpZmljYXRlcyBSb290IENBMB4XDTE5MDgxMzE5MTUw\n" +
		"MloXDTI5MDgxMDE5MTUwMlowJDEiMCAGA1UEAxMZU3RlcCBDZXJ0aWZpY2F0ZXMg\n" +
		"Um9vdCBDQTBZMBMGByqGSM49AgEGCCqGSM49AwEHA0IABAMVL7W0Pm3oJUfI4wXd\n" +
		"klDEnn5XSmj86X0amCA0gcO1tITPmCW3Bpe4pOoWUvZVeQdoScq7znkUt2/G2t1N\n" +
		"71ijRTBDMA4GA1UdDwEB/wQEAwIBBjASBgNVHRMBAf8ECDAGAQH/AgEBMB0GA1Ud\n" +
		"DgQWBBRucPrVnPvZN0r4AU9Lg2/eBrx7kjAKBggqhkjOPQQDAgNIADBFAiBRRAtk\n" +
		"5zLcGhCahmPnW20dLitC3EWMiQ4lDp7aEz+EPAIhAI9fVs5qoItmT8jp6ZKU5Q2u\n" +
		"aDPk8k2CnN27rFsYWupL\n" +
		"-----END CERTIFICATE-----\n")

	if _, err := ca.NewClient("https://ca.example.com", caClientOption(bundle)); err != nil {
		t.Fatalf("expected no error configuring the client with a caBundle, got %v", err)
	}
}
