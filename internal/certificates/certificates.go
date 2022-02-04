package certificates

import (
	"context"
	"crypto/tls"
	"fmt"

	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/infrahq/infra/pki"
)

type Options struct {
}

type CertificateManager struct {
	pki.CertificateProvider
	Options
}

func (c *CertificateManager) LoadCertificates() error {
	if len(c.CertificateProvider.ActiveCAs()) == 0 {
		if err := c.CertificateProvider.CreateCA(); err != nil {
			return fmt.Errorf("creating CA certificates: %w", err)
		}
	}

	// automatically rotate CAs as the oldest one expires
	if len(c.ActiveCAs()) == 1 {
		if err := c.CertificateProvider.RotateCA(); err != nil {
			return fmt.Errorf("rotating CA: %w", err)
		}
	}

	return nil
}

func (c *CertificateManager) Serve(ctx context.Context, tlsConfig *tls.Config) error {
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())

	g := r.Group("/")
	post(g, "/certificatesigningrequest", c.handleCertificateSigningRequest)

	s := &http.Server{
		Addr:      ":https",
		TLSConfig: tlsConfig,
		Handler:   r,
	}

	return s.ListenAndServeTLS("", "")
}

type CertificateSigningRequest struct {
	PublicCertificate []byte `json:"public_certificate"`
}

type CertificateSigningResponse struct {
	SignedCertifiacate []byte `json:"signed_certificate"`
	PendingApproval    bool   `json:"pending_approval"`
}

func (c *CertificateManager) handleCertificateSigningRequest(ctx *gin.Context, req *CertificateSigningRequest) (*CertificateSigningResponse, error) {

	return nil, nil
}
