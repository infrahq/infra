package connector

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"net/http/httputil"
	"sync"

	"github.com/gin-gonic/gin"

	"github.com/infrahq/infra/internal/certs"
	"github.com/infrahq/infra/internal/logging"
)

func proxyMiddleware(
	proxy *httputil.ReverseProxy,
	authn *authenticator,
	bearerToken string,
) gin.HandlerFunc {
	return func(c *gin.Context) {
		claim, err := authn.Authenticate(c.Request)
		if err != nil {
			logging.L.Info().Err(err).Msgf("failed to authenticate request")
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		c.Request.Header.Set("Impersonate-User", claim.Name)
		for _, g := range claim.Groups {
			c.Request.Header.Add("Impersonate-Group", g)
		}

		c.Request.Header.Set("Authorization", fmt.Sprintf("Bearer %s", bearerToken))
		proxy.ServeHTTP(c.Writer, c.Request)
	}
}

type CertCache struct {
	mu     sync.Mutex
	caCert []byte
	caKey  []byte
	hosts  []string
	cert   *tls.Certificate
}

func (c *CertCache) AddHost(host string) (*tls.Certificate, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, h := range c.hosts {
		if h == host {
			return c.cert, nil
		}
	}

	c.hosts = append(c.hosts, host)

	logging.Debugf("generating certificate for: %v", c.hosts)

	ca, err := tls.X509KeyPair(c.caCert, c.caKey)
	if err != nil {
		return nil, err
	}

	caCert, err := x509.ParseCertificate(ca.Certificate[0])
	if err != nil {
		return nil, err
	}

	certPEM, keyPEM, err := certs.GenerateCertificate(c.hosts, caCert, ca.PrivateKey)
	if err != nil {
		return nil, err
	}

	tlsCert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return nil, err
	}

	c.cert = &tlsCert

	return c.cert, nil
}

// readCertificate is a threadsafe way to read the certificate
func (c *CertCache) readCertificate() *tls.Certificate {
	c.mu.Lock()
	cert := c.cert
	c.mu.Unlock()
	return cert
}

// Certificate returns a TLS certificate for the connector, if one does not exist it is generated for the empty host
func (c *CertCache) Certificate() (*tls.Certificate, error) {
	cert := c.readCertificate()
	if cert == nil {
		// the host is not available externally, or this would have been set
		// set to an empty host for the liveness check to resolve from the same host
		return c.AddHost("")
	}

	return cert, nil
}

func NewCertCache(caCertPEM []byte, caKeyPem []byte) *CertCache {
	return &CertCache{caCert: caCertPEM, caKey: caKeyPem}
}
