package pki

import (
	"crypto/tls"
	"crypto/x509"
)

type CertificateProvider interface {
	// A setup step; create a root CA. this happens only once.
	CreateCA() error

	// rotate the current CA. This does a half-rotation. the current cert becomes the previous cert, and there are always two active certificates
	RotateCA() error

	// return the two active CA certificates. This always returns two, and the second one is always the most recent
	ActiveCAs() []x509.Certificate

	// return active CAs as tls certificates, this includes the private keys; it's used for the servers to listen for requests and be able to read the responses.
	TLSCertificates() ([]tls.Certificate, error)

	// Sign a cert with the latest active CA.
	// Caller should have already validated that it's okay to sign this certificate by verifying the sender's authenticity, and that they own the resources they're asking to be certified for.
	// A Certificate Signing Request can be parsed with `x509.ParseCertificateRequest()`
	SignCertificate(csr x509.CertificateRequest) (pemBytes []byte, err error)

	// Preload attempts to preload the root certificate into the system. If this is not possible in this implementation of the certificate provider, it should return internal.ErrNotImplemented or a simple errors.New("not implemented")
	Preload(rootCACertificate, publicKey []byte) error
}
