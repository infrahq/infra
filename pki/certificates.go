package pki

import (
	"crypto/ed25519"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"math/rand"
	"strings"
	"time"

	"github.com/infrahq/infra/internal/server/models"
)

// the pki package defines an interface and implementations of public key encryption, specifically around certificates.

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

func MakeUserCert(commonName string, lifetime time.Duration) (*KeyPair, error) {
	pub, prv, err := ed25519.GenerateKey(randReader)
	if err != nil {
		return nil, fmt.Errorf("generating keys: %w", err)
	}

	certTemplate := x509.Certificate{
		PublicKeyAlgorithm: x509.Ed25519,
		PublicKey:          pub,
		SerialNumber:       big.NewInt(rand.Int63()), //nolint:gosec
		Subject:            pkix.Name{CommonName: commonName},
		NotBefore:          time.Now().Add(-5 * time.Minute).UTC(),
		NotAfter:           time.Now().Add(lifetime).UTC(),
		KeyUsage:           x509.KeyUsageDataEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:        []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}

	rawCert, err := x509.CreateCertificate(randReader, &certTemplate, &certTemplate, pub, prv)
	if err != nil {
		return nil, fmt.Errorf("creating certificate: %w", err)
	}

	cert, err := x509.ParseCertificate(rawCert)
	if err != nil {
		return nil, fmt.Errorf("parsing self-created certificate: %w", err)
	}

	pemBytes := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: rawCert,
	})

	keyPair := &KeyPair{
		Cert:       cert,
		CertPEM:    pemBytes,
		PublicKey:  pub,
		PrivateKey: prv,
	}

	return keyPair, nil
}

func SignUserCert(cp CertificateProvider, cert *x509.Certificate, user *models.Identity) (*x509.Certificate, []byte, error) {
	if len(cert.Raw) == 0 {
		panic("cert.Raw is missing")
	}

	rawCert := cert.Raw

	if !strings.HasPrefix(cert.Subject.CommonName, "User ") {
		return nil, nil, fmt.Errorf("invalid certificate common name for user certificate")
	}

	pem1, err := cp.SignCertificate(x509.CertificateRequest{
		Raw:                rawCert,
		PublicKeyAlgorithm: cert.PublicKeyAlgorithm,
		PublicKey:          cert.PublicKey,
		Subject:            cert.Subject,
		EmailAddresses:     []string{user.Name},
		Extensions:         cert.Extensions,
		ExtraExtensions:    cert.ExtraExtensions,
		SignatureAlgorithm: x509.PureEd25519,
	})
	if err != nil {
		return nil, nil, err
	}

	p, rest := pem.Decode(pem1)
	if p == nil {
		return nil, nil, fmt.Errorf("decoding certificate: %w", err)
	}

	if len(rest) > 0 {
		panic("forgot part of cert chain")
	}

	newCert, err := x509.ParseCertificate(p.Bytes)
	if err != nil {
		return nil, nil, fmt.Errorf("parsing certificate: %w", err)
	}

	return newCert, pem1, nil
}
