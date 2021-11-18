package pki

import "crypto/x509"

// the pki package defines an interface and implementations of public key encryption, specifically around certificates.

type CertificateProvider interface {
	// A setup step; create a root CA. this happens only once.
	CreateCA() error
	// rotate the current CA. This does a half-rotation. the current cert becomes the previous cert, and there are always two active certificates
	RotateCA() error
	// return the two active CA certificates. This always returns two, and the second one is always the most recent
	ActiveCAs() []x509.Certificate
	// return the chain of certificates, active or not, since x
	// CertificateChain() error
	// IssueCert() // don't think I need this

	// Sign a cert with the latest active CA.
	// Caller should have already validated that it's okay to sign this certificate by verifying the sender's authenticity, and that they own the resources they're asking to be certified for.
	// A Certificate Signing Request can be parsed with `x509.ParseCertificateRequest()`
	SignCertificate(csr x509.CertificateRequest) (pemBytes []byte, err error)
}

// type Signer interface {
// 	SignCert()
// }

func ValidateCertificate() {}
