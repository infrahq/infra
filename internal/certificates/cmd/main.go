package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"

	"github.com/infrahq/infra/internal/certificates"
	"github.com/infrahq/infra/pki"
)

// Run starts the http listener and binds it to the certificate manager
func Run(options *certificates.Options) error {
	certProvider, err := pki.NewNativeCertificateProvider(pki.NativeCertificateProviderConfig{
		// TODO: temporary
		StoragePath:                   "/var/tmp/certs",
		FullKeyRotationDurationInDays: 365,
	})
	if err != nil {
		return fmt.Errorf("creating cert provider: %w", err)
	}

	cm := &certificates.CertificateManager{
		Options:             *options,
		CertificateProvider: certProvider,
	}

	if err = cm.LoadCertificates(); err != nil {
		return fmt.Errorf("loading certificates: %w", err)
	}

	clientCACertPool := x509.NewCertPool()
	for _, cert := range cm.CertificateProvider.ActiveCAs() {
		clientCACertPool.AddCert(&cert)
	}

	tlsCertificates, err := cm.CertificateProvider.TLSCertificates()
	if err != nil {
		return fmt.Errorf("parsing tls certificates: %w", err)
	}

	tlsConfig := &tls.Config{
		// certificates to present to client
		Certificates: tlsCertificates,
		MinVersion:   tls.VersionTLS13, // if we drop this to 1.2 we may need to exclude some ciphers. *narrows eyes at RSA*
		// VerifyPeerCertificate: ,
		// VerifyConnection: ,
		ClientAuth: tls.RequireAndVerifyClientCert,
		ClientCAs:  clientCACertPool,
	}

	ctx := context.Background()

	return cm.Serve(ctx, tlsConfig)
}

func main() {
	options := &certificates.Options{}
	Run(options)
}
