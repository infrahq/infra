package pki

import (
	"crypto/ed25519"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
)

type KeyPair struct {
	KeyType       string // KeyType //  (string) == "ed25519" maybe
	PublicKey     ed25519.PublicKey
	PrivateKey    ed25519.PrivateKey `json:",omitempty"`
	CertRaw       []byte             `json:",omitempty"`
	SignedCertRaw []byte             `json:",omitempty"`
	Cert          *x509.Certificate  `json:"-"`
	SignedCert    *x509.Certificate  `json:"-"`
}

func (k *KeyPair) TLSCertificate() (*tls.Certificate, error) {
	bytes := k.SignedCertRaw
	if len(bytes) == 0 {
		bytes = k.CertRaw
	}

	keyPEM, err := MarshalPrivateKey(k.PrivateKey)

	cert, err := tls.X509KeyPair(bytes, keyPEM)
	if err != nil {
		return nil, fmt.Errorf("reading keypair: %w", err)
	}

	return &cert, nil
}

func (k *KeyPair) UnmarshalJSON(data []byte) error {
	type TmpKeyPair KeyPair
	tmpKeyPair := &TmpKeyPair{}

	err := json.Unmarshal(data, &tmpKeyPair)
	if err != nil {
		return err
	}

	k.PublicKey = tmpKeyPair.PublicKey
	k.PrivateKey = tmpKeyPair.PrivateKey
	k.CertRaw = tmpKeyPair.CertRaw
	k.SignedCertRaw = tmpKeyPair.SignedCertRaw

	p, _ := pem.Decode(k.CertRaw)
	cert, err := x509.ParseCertificate(p.Bytes)
	if err != nil {
		return fmt.Errorf("parsing raw certificate: %w", err)
	}

	k.Cert = cert

	if len(k.SignedCertRaw) > 0 {
		p, _ = pem.Decode(k.SignedCertRaw)
		cert, err := x509.ParseCertificate(p.Bytes)
		if err != nil {
			return fmt.Errorf("parsing signed certificate: %w", err)
		}

		k.SignedCert = cert
	}

	return nil
}
