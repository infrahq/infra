package certificates

import (
	"crypto/tls"
	"crypto/x509"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type mockCertificateProvider struct {
	mainCA x509.Certificate
	prevCA x509.Certificate
}

func (m *mockCertificateProvider) CreateCA() error {
	return nil
}
func (m *mockCertificateProvider) RotateCA() error {
	return nil
}
func (m *mockCertificateProvider) ActiveCAs() []x509.Certificate {
	return []x509.Certificate{}
}
func (m *mockCertificateProvider) TLSCertificates() ([]tls.Certificate, error) {
	return []tls.Certificate{}, nil
}
func (m *mockCertificateProvider) SignCertificate(csr x509.CertificateRequest) (pemBytes []byte, err error) {
	return nil, nil
}

func TestGetCertificateSigningRequest(t *testing.T) {
	cm := &CertificateManager{
		CertificateProvider: &mockCertificateProvider{},
	}

	c, _ := gin.CreateTestContext(nil)
	resp, err := cm.handleCertificateSigningRequest(c, &CertificateSigningRequest{})
	require.NoError(t, err)

	require.Equal(t, true, resp.PendingApproval)
}

func TestGetFingerprintFromKubernetes(t *testing.T) {}

func TestSignCertificate(t *testing.T) {}

func TestGetActiveCertificates(t *testing.T) {}

func TestCreateRootCA(t *testing.T) {}

func TestRotateCA(t *testing.T) {}

func TestTrustFingerprint(t *testing.T) {}
