package pki

import (
	"crypto"
	"crypto/ed25519"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
)

// PrivateKey is an interface compatible with the go stdlib ed25519, ecdsa, and rsa keys
type PrivateKey interface {
	Public() crypto.PublicKey
	Equal(x crypto.PrivateKey) bool
}

type KeyPair struct {
	KeyAlgorithm     string
	SigningAlgorithm string
	PublicKey        ed25519.PublicKey
	PrivateKey       ed25519.PrivateKey `json:",omitempty"`
	CertPEM          []byte             `json:",omitempty"` // pem encoded, does not contain private key
	SignedCertPEM    []byte             `json:",omitempty"` // pem encoded, does not contain private key
	Cert             *x509.Certificate  `json:"-"`
	SignedCert       *x509.Certificate  `json:"-"`
}

func (k *KeyPair) TLSCertificate() (*tls.Certificate, error) {
	bytes := k.SignedCertPEM
	if len(bytes) == 0 {
		bytes = k.CertPEM
	}

	keyPEM, err := MarshalPrivateKey(k.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("marshal keypair: %w", err)
	}

	cert, err := tls.X509KeyPair(bytes, keyPEM)
	if err != nil {
		return nil, fmt.Errorf("reading keypair: %w", err)
	}

	return &cert, nil
}

func (k *KeyPair) UnmarshalJSON(data []byte) error {
	type TemporaryKeyPair KeyPair

	tmpKeyPair := &TemporaryKeyPair{}

	err := json.Unmarshal(data, &tmpKeyPair)
	if err != nil {
		return err
	}

	k.PublicKey = tmpKeyPair.PublicKey
	k.PrivateKey = tmpKeyPair.PrivateKey
	k.CertPEM = tmpKeyPair.CertPEM
	k.SignedCertPEM = tmpKeyPair.SignedCertPEM

	p, _ := pem.Decode(k.CertPEM)

	cert, err := x509.ParseCertificate(p.Bytes)
	if err != nil {
		return fmt.Errorf("parsing raw certificate: %w", err)
	}

	k.Cert = cert

	if len(k.SignedCertPEM) > 0 {
		p, _ = pem.Decode(k.SignedCertPEM)

		cert, err := x509.ParseCertificate(p.Bytes)
		if err != nil {
			return fmt.Errorf("parsing signed certificate: %w", err)
		}

		k.SignedCert = cert
	}

	return nil
}
