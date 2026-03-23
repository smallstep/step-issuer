package provisioners

import (
	"bytes"
	"context"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"

	certmanager "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	capi "github.com/smallstep/certificates/api"
	"github.com/smallstep/certificates/ca"
	api "github.com/smallstep/step-issuer/api/v1beta1"
	"k8s.io/apimachinery/pkg/types"
)

var collection = new(sync.Map)

// CustomHeaderTransport is an HTTP transport that injects a custom header into requests.
type CustomHeaderTransport struct {
	transport http.RoundTripper
	header    *api.CustomHeader
}

// RoundTrip implements the http.RoundTripper interface.
func (t *CustomHeaderTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	if t.header != nil {
		value, err := readHeaderValue(t.header.Value)
		if err != nil {
			return nil, err
		}
		if value != "" {
			r.Header.Set(t.header.Name, value)
		}
	}
	return t.transport.RoundTrip(r)
}

// NewCustomHeaderTransport creates a new transport that injects custom headers.
func NewCustomHeaderTransport(transport http.RoundTripper, header *api.CustomHeader) *CustomHeaderTransport {
	if transport == nil {
		transport = http.DefaultTransport
	}
	return &CustomHeaderTransport{
		transport: transport,
		header:    header,
	}
}

// Step implements a Step JWK provisioners in charge of signing certificate
// requests using step certificates.
type Step struct {
	name         string
	caBundle     []byte
	provisioner  *ca.Provisioner
	customHeader *api.CustomHeader
}

// NewFromStepIssuer returns a new Step provisioner, configured with the information in the
// given issuer.
func NewFromStepIssuer(iss *api.StepIssuer, password []byte) (*Step, error) {
	options := []ca.ClientOption{
		ca.WithCABundle(iss.Spec.CABundle),
	}

	// Add custom HTTP transport if custom header is configured
	if iss.Spec.CustomHeader != nil {
		options = append(options, ca.WithTransportDecorator(func(tr http.RoundTripper) http.RoundTripper {
			return NewCustomHeaderTransport(tr, iss.Spec.CustomHeader)
		}))
	}

	provisioner, err := ca.NewProvisioner(iss.Spec.Provisioner.Name, iss.Spec.Provisioner.KeyID, iss.Spec.URL, password, options...)
	if err != nil {
		return nil, err
	}

	p := &Step{
		name:         iss.Name + "." + iss.Namespace,
		caBundle:     iss.Spec.CABundle,
		provisioner:  provisioner,
		customHeader: iss.Spec.CustomHeader,
	}

	return p, nil
}

func NewFromStepClusterIssuer(iss *api.StepClusterIssuer, password []byte) (*Step, error) {
	options := []ca.ClientOption{
		ca.WithCABundle(iss.Spec.CABundle),
	}

	// Add custom HTTP transport if custom header is configured
	if iss.Spec.CustomHeader != nil {
		options = append(options, ca.WithTransportDecorator(func(tr http.RoundTripper) http.RoundTripper {
			return NewCustomHeaderTransport(tr, iss.Spec.CustomHeader)
		}))
	}

	provisioner, err := ca.NewProvisioner(iss.Spec.Provisioner.Name, iss.Spec.Provisioner.KeyID, iss.Spec.URL, password, options...)
	if err != nil {
		return nil, err
	}

	p := &Step{
		name:         iss.Name + "." + iss.Namespace,
		caBundle:     iss.Spec.CABundle,
		provisioner:  provisioner,
		customHeader: iss.Spec.CustomHeader,
	}

	return p, nil
}

// Load returns a Step provisioner by NamespacedName.
func Load(namespacedName types.NamespacedName) (*Step, bool) {
	v, ok := collection.Load(namespacedName)
	if !ok {
		return nil, ok
	}
	p, ok := v.(*Step)
	return p, ok
}

// Store adds a new provisioner to the collection by NamespacedName.
func Store(namespacedName types.NamespacedName, provisioner *Step) {
	collection.Store(namespacedName, provisioner)
}

// Sign sends the certificate requests to the Step CA and returns the signed
// certificate.
func (s *Step) Sign(_ context.Context, cr *certmanager.CertificateRequest) ([]byte, []byte, error) {
	// decode and check certificate request
	csr, err := decodeCSR(cr.Spec.Request)
	if err != nil {
		return nil, nil, err
	}

	sans := append([]string{}, csr.DNSNames...)
	sans = append(sans, csr.EmailAddresses...)
	for _, ip := range csr.IPAddresses {
		sans = append(sans, ip.String())
	}
	for _, u := range csr.URIs {
		sans = append(sans, u.String())
	}

	subject := csr.Subject.CommonName
	if subject == "" {
		subject = generateSubject(sans)
	}

	token, err := s.provisioner.Token(subject, sans...)
	if err != nil {
		return nil, nil, err
	}

	var notAfter capi.TimeDuration
	if cr.Spec.Duration != nil {
		notAfter.SetDuration(cr.Spec.Duration.Duration)
	}

	resp, err := s.provisioner.Sign(&capi.SignRequest{
		CsrPEM: capi.CertificateRequest{
			CertificateRequest: csr,
		},
		OTT:      token,
		NotAfter: notAfter,
	})
	if err != nil {
		return nil, nil, err
	}

	// Encode server certificate with the intermediate
	chainPem, err := encodeX509(resp.CertChainPEM...)
	if err != nil {
		return nil, nil, err
	}
	return chainPem, s.caBundle, nil
}

// readHeaderValue reads the custom header value, handling both static values
// and file:// URIs. Returns the header value to use, or empty string if value is empty.
func readHeaderValue(value string) (string, error) {
	if value == "" {
		return "", nil
	}

	// Check if value is a file:// URI
	if strings.HasPrefix(value, "file://") {
		filePath := strings.TrimPrefix(value, "file://")
		content, err := os.ReadFile(filePath)
		if err != nil {
			return "", fmt.Errorf("failed to read file %s: %w", filePath, err)
		}
		return string(bytes.TrimSpace(content)), nil
	}

	// Static value - return as-is
	return value, nil
}

// decodeCSR decodes a certificate request in PEM format and returns the
func decodeCSR(data []byte) (*x509.CertificateRequest, error) {
	block, rest := pem.Decode(data)
	if block == nil || len(rest) > 0 {
		return nil, fmt.Errorf("unexpected CSR PEM on sign request")
	}
	if block.Type != "CERTIFICATE REQUEST" {
		return nil, fmt.Errorf("PEM is not a certificate request")
	}
	csr, err := x509.ParseCertificateRequest(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("error parsing certificate request: %w", err)
	}
	if err := csr.CheckSignature(); err != nil {
		return nil, fmt.Errorf("error checking certificate request signature: %w", err)
	}
	return csr, nil
}

// encodeX509 will encode a certificate into PEM format.
func encodeX509(certs ...capi.Certificate) ([]byte, error) {
	certPem := bytes.NewBuffer([]byte{})
	for _, cert := range certs {
		err := pem.Encode(certPem, &pem.Block{Type: "CERTIFICATE", Bytes: cert.Raw})
		if err != nil {
			return nil, err
		}
	}
	return certPem.Bytes(), nil
}

// generateSubject returns the first SAN that is not 127.0.0.1 or localhost. The
// CSRs generated by the Certificate resource have always those SANs. If no SANs
// are available `step-issuer-certificate` will be used as a subject is always
// required.
func generateSubject(sans []string) string {
	if len(sans) == 0 {
		return "step-issuer-certificate"
	}
	for _, s := range sans {
		if s != "127.0.0.1" && s != "localhost" {
			return s
		}
	}
	return sans[0]
}
