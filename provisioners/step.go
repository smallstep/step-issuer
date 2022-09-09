package provisioners

import (
	"bytes"
	"context"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"sync"

	certmanager "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1"
	capi "github.com/smallstep/certificates/api"
	"github.com/smallstep/certificates/ca"
	api "github.com/smallstep/step-issuer/api/v1beta1"
	"k8s.io/apimachinery/pkg/types"
)

var collection = new(sync.Map)

// Step implements a Step JWK provisioners in charge of signing certificate
// requests using step certificates.
type Step struct {
	name        string
	provisioner *ca.Provisioner
}

// NewFromStepIssuer returns a new Step provisioner, configured with the information in the
// given issuer.
func NewFromStepIssuer(iss *api.StepIssuer, password []byte) (*Step, error) {
	var options []ca.ClientOption
	if len(iss.Spec.CABundle) > 0 {
		options = append(options, ca.WithCABundle(iss.Spec.CABundle))
	}
	provisioner, err := ca.NewProvisioner(iss.Spec.Provisioner.Name, iss.Spec.Provisioner.KeyID, iss.Spec.URL, password, options...)
	if err != nil {
		return nil, err
	}

	p := &Step{
		name:        iss.Name + "." + iss.Namespace,
		provisioner: provisioner,
	}

	// Request identity certificate if required.
	if version, err := provisioner.Version(); err == nil {
		if version.RequireClientAuthentication {
			if err := p.createIdentityCertificate(); err != nil {
				return nil, err
			}
		}
	}

	return p, nil
}

func NewFromStepClusterIssuer(iss *api.StepClusterIssuer, password []byte) (*Step, error) {
	var options []ca.ClientOption
	if len(iss.Spec.CABundle) > 0 {
		options = append(options, ca.WithCABundle(iss.Spec.CABundle))
	}
	provisioner, err := ca.NewProvisioner(iss.Spec.Provisioner.Name, iss.Spec.Provisioner.KeyID, iss.Spec.URL, password, options...)
	if err != nil {
		return nil, err
	}

	p := &Step{
		name:        iss.Name + "." + iss.Namespace,
		provisioner: provisioner,
	}

	// Request identity certificate if required.
	if version, err := provisioner.Version(); err == nil {
		if version.RequireClientAuthentication {
			if err := p.createIdentityCertificate(); err != nil {
				return nil, err
			}
		}
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

func (s *Step) createIdentityCertificate() error {
	csr, pk, err := ca.CreateCertificateRequest(s.name)
	if err != nil {
		return err
	}
	token, err := s.provisioner.Token(s.name)
	if err != nil {
		return err
	}
	resp, err := s.provisioner.Sign(&capi.SignRequest{
		CsrPEM: *csr,
		OTT:    token,
	})
	if err != nil {
		return err
	}
	tr, err := s.provisioner.Client.Transport(context.Background(), resp, pk)
	if err != nil {
		return err
	}
	s.provisioner.Client.SetTransport(tr)
	return nil
}

// Sign sends the certificate requests to the Step CA and returns the signed
// certificate.
func (s *Step) Sign(ctx context.Context, cr *certmanager.CertificateRequest) ([]byte, []byte, error) {
	// Get root certificate(s)
	roots, err := s.provisioner.Roots()
	if err != nil {
		return nil, nil, err
	}

	// Encode root certificates
	caPem, err := encodeX509(roots.Certificates...)
	if err != nil {
		return nil, nil, err
	}

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
	return chainPem, caPem, nil
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
