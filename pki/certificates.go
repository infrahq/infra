package pki

import (
	"crypto/ed25519"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"math/rand"
	"net"
	"time"

	"github.com/infrahq/infra/internal/server/models"
)

// the pki package defines an interface and implementations of public key encryption, specifically around certificates.

func MakeConnectorCert(hosts []string, lifetime time.Duration) (*KeyPair, error) {
	pub, prv, err := ed25519.GenerateKey(randReader)
	if err != nil {
		return nil, fmt.Errorf("generating keys: %w", err)
	}

	certTemplate := x509.Certificate{
		SerialNumber:       big.NewInt(rand.Int63()), //nolint:gosec
		PublicKeyAlgorithm: x509.Ed25519,
		PublicKey:          pub,
		Subject: pkix.Name{
			Organization: []string{"Infra"},
		},
		NotBefore: time.Now().Add(-5 * time.Minute).UTC(),
		NotAfter:  time.Now().Add(lifetime).UTC(),
		KeyUsage: x509.KeyUsageKeyEncipherment |
			x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}

	for _, h := range hosts {
		if ip := net.ParseIP(h); ip != nil {
			certTemplate.IPAddresses = append(certTemplate.IPAddresses, ip)
		} else {
			certTemplate.DNSNames = append(certTemplate.DNSNames, h)
		}
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

func MakeUserCert(email string, lifetime time.Duration) (*KeyPair, error) {
	pub, prv, err := ed25519.GenerateKey(randReader)
	if err != nil {
		return nil, fmt.Errorf("generating keys: %w", err)
	}

	certTemplate := x509.Certificate{
		PublicKeyAlgorithm: x509.Ed25519,
		PublicKey:          pub,
		SerialNumber:       big.NewInt(rand.Int63()), //nolint:gosec
		EmailAddresses:     []string{email},
		NotBefore:          time.Now().Add(-5 * time.Minute).UTC(),
		NotAfter:           time.Now().Add(lifetime).UTC(),
		KeyUsage: x509.KeyUsageKeyEncipherment |
			x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
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

func SignConnectorCert(cp CertificateProvider, cert *x509.Certificate) (*x509.Certificate, []byte, error) {
	if len(cert.Raw) == 0 {
		panic("cert.Raw is missing")
	}

	rawCert := cert.Raw

	cert.Subject.CommonName = "Connector"
	pem1, err := cp.SignCertificate(x509.CertificateRequest{
		Raw:                rawCert,
		PublicKeyAlgorithm: cert.PublicKeyAlgorithm,
		PublicKey:          cert.PublicKey,
		Subject:            cert.Subject,
		DNSNames:           cert.DNSNames,
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

func SignUserCert(cp CertificateProvider, cert *x509.Certificate, user *models.Identity) (*x509.Certificate, []byte, error) {
	if len(cert.Raw) == 0 {
		panic("cert.Raw is missing")
	}

	rawCert := cert.Raw

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
