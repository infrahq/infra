package pki

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"net"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
)

const (
	day        = 24 * time.Hour
	rootCAName = "Root Infra CA"
)

var (
	randReader = rand.Reader

	allowedSignatureAlgorithms = []x509.SignatureAlgorithm{
		x509.ECDSAWithSHA512,
		x509.PureEd25519,
	}
	allowedPublicKeyAlgorithms = []x509.PublicKeyAlgorithm{
		x509.ECDSA,
		x509.Ed25519,
	}
)

type NativeCertificateProvider struct {
	NativeCertificateProviderConfig

	db *gorm.DB

	activeKeypair   KeyPair
	previousKeypair KeyPair
}

type NativeCertificateProviderConfig struct {
	FullKeyRotationDurationInDays int
	KeyAlgorithm                  string // only ed25519 so far.
	SigningAlgorithm              string
	InitialRootCAPublicKey        []byte
	InitialRootCACert             []byte
	InitialRootCAPrivateKey       []byte
}

func NewNativeCertificateProvider(db *gorm.DB, cfg NativeCertificateProviderConfig) (*NativeCertificateProvider, error) {
	if cfg.FullKeyRotationDurationInDays == 0 {
		cfg.FullKeyRotationDurationInDays = 365
	}

	p := &NativeCertificateProvider{
		NativeCertificateProviderConfig: cfg,
		db:                              db,
	}

	if err := p.loadFromDB(); err != nil {
		return nil, err
	}

	if p.activeKeypair.SignedCert == nil &&
		len(cfg.InitialRootCAPublicKey) > 0 &&
		len(cfg.InitialRootCACert) > 0 &&
		len(cfg.InitialRootCAPrivateKey) > 0 {
		pubKey, err := base64.StdEncoding.DecodeString(string(cfg.InitialRootCAPublicKey))
		if err != nil {
			return nil, fmt.Errorf("reading initialRootCAPublicKey: %w", err)
		}

		cert, err := base64.StdEncoding.DecodeString(string(cfg.InitialRootCACert))
		if err != nil {
			return nil, fmt.Errorf("reading initialRootCACert: %w", err)
		}

		prvKey, err := base64.StdEncoding.DecodeString(string(cfg.InitialRootCAPrivateKey))
		if err != nil {
			return nil, fmt.Errorf("reading initialRootCAPrivateKey: %w", err)
		}

		c, err := x509.ParseCertificate(cert)
		if err != nil {
			return nil, fmt.Errorf("parsing initialRootCACert: %w", err)
		}

		certPEM := pem.EncodeToMemory(&pem.Block{
			Type:  "CERTIFICATE",
			Bytes: cert,
		})

		p.activeKeypair = KeyPair{
			KeyAlgorithm:     p.KeyAlgorithm,
			SigningAlgorithm: p.SigningAlgorithm,
			PublicKey:        pubKey,
			PrivateKey:       prvKey,
			SignedCertPEM:    certPEM,
			SignedCert:       c,
		}
	}

	return p, nil
}

func (n *NativeCertificateProvider) Preload(rootCACertificate, publicKey []byte) (err error) {
	if n.activeKeypair.SignedCert != nil {
		return fmt.Errorf("cannot preload a certificate when another one is already loaded.")
	}

	partsFound := 0
	rest := rootCACertificate

	var (
		p          *pem.Block
		cert       *x509.Certificate
		privateKey ed25519.PrivateKey
	)

	for len(rest) > 0 {
		partsFound++

		p, rest = pem.Decode(rootCACertificate)

		switch {
		case strings.Contains(p.Type, "PRIVATE KEY"):
			key, err := x509.ParsePKCS8PrivateKey(p.Bytes)
			if err != nil {
				return fmt.Errorf("parsing private key from certificate: %w", err)
			}

			var ok bool

			privateKey, ok = key.(ed25519.PrivateKey)
			if !ok {
				return fmt.Errorf("unknown type for key: %T", key)
			}

		case strings.Contains(p.Type, "CERTIFICATE"):
			cert, err = x509.ParseCertificate(p.Bytes)
			if err != nil {
				return fmt.Errorf("parsing root certificate: %w", err)
			}
		}
	}

	if partsFound > 2 {
		return fmt.Errorf("expected one certificate and one private key, but got certificate chain")
	}

	if partsFound < 2 {
		return fmt.Errorf("expected one certificate and one private key")
	}

	n.activeKeypair = KeyPair{
		KeyAlgorithm:     cert.PublicKeyAlgorithm.String(),
		SigningAlgorithm: cert.SignatureAlgorithm.String(),
		PublicKey:        publicKey,
		PrivateKey:       privateKey,
		SignedCertPEM:    rootCACertificate,
		SignedCert:       cert,
	}

	return n.RotateCA()
}

// CreateCA creates a new root CA and immediately does a half-rotation.
// the new active key after rotation is the one that should be used.
func (n *NativeCertificateProvider) CreateCA() error {
	pub, prv, err := ed25519.GenerateKey(randReader)
	if err != nil {
		return fmt.Errorf("generating keys: %w", err)
	}

	n.activeKeypair.PrivateKey = prv
	n.activeKeypair.PublicKey = pub

	validFor := time.Duration(n.FullKeyRotationDurationInDays/2) * day

	cert, raw, err := createCertSignedBy(n.activeKeypair, n.activeKeypair, validFor)
	if err != nil {
		return err
	}

	pemBytes := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: raw,
	})

	n.activeKeypair.SignedCertPEM = pemBytes
	n.activeKeypair.SignedCert = cert
	n.activeKeypair.KeyAlgorithm = x509.Ed25519.String()
	n.activeKeypair.SigningAlgorithm = x509.PureEd25519.String()

	return n.RotateCA()
}

// ActiveCAs returns the currently in-use CAs, the newest cert is always the last in the list
func (n *NativeCertificateProvider) ActiveCAs() []x509.Certificate {
	result := []x509.Certificate{}

	if n.previousKeypair.SignedCert != nil && certActive(n.previousKeypair.SignedCert) {
		result = append(result, *n.previousKeypair.SignedCert)
	}

	if n.activeKeypair.SignedCert != nil && certActive(n.activeKeypair.SignedCert) {
		result = append(result, *n.activeKeypair.SignedCert)
	}

	return result
}

func certActive(cert *x509.Certificate) bool {
	if cert.NotBefore.After(time.Now()) {
		return false
	}

	if cert.NotAfter.Before(time.Now()) {
		return false
	}

	return true
}

// TODO: SignCertificate should be renamed to SignUserCertificate?
func (n *NativeCertificateProvider) SignCertificate(csr x509.CertificateRequest) (pemBytes []byte, err error) {
	switch {
	case csr.Subject.CommonName == rootCAName:
		return nil, fmt.Errorf("cannot sign cert pretending to be the root CA")
	case strings.HasPrefix(csr.Subject.CommonName, "Connector"):
	case strings.HasPrefix(csr.Subject.CommonName, "Infra Server"):
	case strings.HasPrefix(csr.Subject.CommonName, "User"):
		// these are ok.
	default:
		return nil, fmt.Errorf("invalid Subject name %q", csr.Subject.CommonName)
	}

	if !isAllowedSignatureAlgorithm(csr.SignatureAlgorithm) {
		return nil, fmt.Errorf("%q is not an acceptable signature algorithm, expecting one of: %v", csr.SignatureAlgorithm, allowedSignatureAlgorithms)
	}

	if !isAllowedPublicKeyAlgorithm(csr.PublicKeyAlgorithm) {
		return nil, fmt.Errorf("%q is not an acceptable public key algorithm, expecting one of: %v", csr.PublicKeyAlgorithm, allowedPublicKeyAlgorithms)
	}

	certTemplate := &x509.Certificate{
		Signature:          csr.Signature,
		SignatureAlgorithm: csr.SignatureAlgorithm,
		PublicKeyAlgorithm: csr.PublicKeyAlgorithm,
		PublicKey:          csr.PublicKey,
		SerialNumber:       big.NewInt(2),
		Issuer:             n.activeKeypair.SignedCert.Subject,
		Subject:            csr.Subject,
		EmailAddresses:     csr.EmailAddresses,
		NotBefore:          time.Now().Add(-5 * time.Minute).UTC(),
		NotAfter:           time.Now().Add(24 * time.Hour).UTC(),
		KeyUsage:           x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:        []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
	}

	if !n.activeKeypair.SignedCert.IsCA {
		panic("not ca")
	}

	if n.activeKeypair.SignedCert.KeyUsage&x509.KeyUsageCertSign != x509.KeyUsageCertSign {
		panic("can't sign keys with this cert")
	}

	signedCert, err := x509.CreateCertificate(randReader, certTemplate, n.activeKeypair.SignedCert, csr.PublicKey, n.activeKeypair.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("creating cert: %w", err)
	}

	pemBytes = pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: signedCert,
	})

	return pemBytes, nil
}

// RotateCA does a half-rotation. the current cert becomes the previous cert, and there are always two active certificates
func (n *NativeCertificateProvider) RotateCA() error {
	n.previousKeypair = n.activeKeypair
	n.activeKeypair = KeyPair{}

	pub, prv, err := ed25519.GenerateKey(randReader)
	if err != nil {
		return fmt.Errorf("generating keys: %w", err)
	}

	n.activeKeypair.PrivateKey = prv
	n.activeKeypair.PublicKey = pub

	validFor := time.Duration(n.FullKeyRotationDurationInDays) * day

	cert, raw, err := createCertSignedBy(n.previousKeypair, n.activeKeypair, validFor)
	if err != nil {
		return err
	}

	pemBytes := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: raw,
	})

	n.activeKeypair.SignedCertPEM = pemBytes
	n.activeKeypair.SignedCert = cert
	n.activeKeypair.KeyAlgorithm = x509.Ed25519.String()
	n.activeKeypair.SigningAlgorithm = x509.PureEd25519.String()

	return n.saveToDB()
}

// createCertSignedBy signs the signee public key using the signer private key, allowing anyone to verify the signature with the signer public key. Certificate expires after _lifetime_
func createCertSignedBy(signer, signee KeyPair, lifetime time.Duration) (*x509.Certificate, []byte, error) {
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)

	serial, err := rand.Int(randReader, serialNumberLimit)
	if err != nil {
		return nil, nil, fmt.Errorf("creating random serial: %w", err)
	}

	certTemplate := &x509.Certificate{
		SignatureAlgorithm: x509.PureEd25519,
		PublicKeyAlgorithm: x509.Ed25519,
		PublicKey:          signee.PublicKey,
		SerialNumber:       serial,
		Issuer:             pkix.Name{CommonName: rootCAName},
		Subject:            pkix.Name{CommonName: rootCAName},
		NotBefore:          time.Now().Add(-5 * time.Minute).UTC(),
		NotAfter:           time.Now().Add(lifetime).UTC(),
		KeyUsage: x509.KeyUsageCertSign |
			x509.KeyUsageDigitalSignature |
			x509.KeyUsageCRLSign |
			x509.KeyUsageKeyAgreement |
			x509.KeyUsageKeyEncipherment,
		ExtKeyUsage: []x509.ExtKeyUsage{
			x509.ExtKeyUsageClientAuth,
			x509.ExtKeyUsageServerAuth,
		},
		IsCA:                  true,
		BasicConstraintsValid: true,

		// SubjectAltName values
		IPAddresses: []net.IP{net.ParseIP("127.0.0.1"), net.IPv6loopback},
		DNSNames:    []string{"localhost"}, // TODO: Support domain names for services?
	}

	signeeCert := signee.Cert
	if signeeCert == nil {
		signeeCert = certTemplate
	}

	if !signeeCert.IsCA {
		return nil, nil, fmt.Errorf("signee cert is not a CA")
	}

	// create client certificate from template and CA public key
	rawCert, err := x509.CreateCertificate(rand.Reader, certTemplate, signeeCert, signee.PublicKey, signee.PrivateKey)
	if err != nil {
		return nil, nil, fmt.Errorf("creating certificate: %w", err)
	}

	cert, err := x509.ParseCertificate(rawCert)
	if err != nil {
		return nil, nil, fmt.Errorf("parsing self-created certificate: %w", err)
	}

	if !cert.IsCA {
		return nil, nil, fmt.Errorf("signed cert is not a CA?")
	}

	return cert, rawCert, nil
}

func (n *NativeCertificateProvider) loadFromDB() error {
	certs, err := data.ListRootCertificates(n.db)
	if err != nil {
		return err
	}

	if len(certs) >= 1 {
		n.activeKeypair, err = certificateToKeyPair(&certs[0])
		if err != nil {
			return err
		}

		if len(certs) >= 2 {
			n.previousKeypair, err = certificateToKeyPair(&certs[1])
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func certificateToKeyPair(c *models.RootCertificate) (KeyPair, error) {
	// the certificate doesn't have pem armoring on it.
	cert, err := x509.ParseCertificate([]byte(c.SignedCert))
	if err != nil {
		return KeyPair{}, fmt.Errorf("couldn't read certificate from db: %w", err)
	}

	return KeyPair{
		KeyAlgorithm:     c.KeyAlgorithm,
		SigningAlgorithm: c.SigningAlgorithm,
		PublicKey:        ed25519.PublicKey(c.PublicKey),
		PrivateKey:       ed25519.PrivateKey(c.PrivateKey),
		SignedCertPEM:    []byte(c.SignedCert),
		SignedCert:       cert,
	}, nil
}

func keyPairToCertificate(k KeyPair) *models.RootCertificate {
	// don't store the certificate with pem encoding; it's padding that only assists a known-plaintext attack
	b, _ := pem.Decode(k.SignedCertPEM)

	return &models.RootCertificate{
		KeyAlgorithm:     k.KeyAlgorithm,
		SigningAlgorithm: k.SigningAlgorithm,
		PublicKey:        models.Base64(k.PublicKey),
		PrivateKey:       models.EncryptedAtRest(k.PrivateKey),
		SignedCert:       models.EncryptedAtRest(b.Bytes),
		ExpiresAt:        k.SignedCert.NotAfter,
	}
}

// saveToDB stores new certs to the database. Used when rotating keys.
func (n *NativeCertificateProvider) saveToDB() error {
	certs := []*models.RootCertificate{
		keyPairToCertificate(n.previousKeypair),
		keyPairToCertificate(n.activeKeypair),
	}
	// only create the previous keypair if it doesn't already exist.
	for _, cert := range certs {
		c, err := data.GetRootCertificate(n.db, data.ByPublicKey(cert.PublicKey))
		if c != nil {
			continue
		}

		if !errors.Is(err, internal.ErrNotFound) {
			return fmt.Errorf("checking for existing cert: %w", err)
		}

		if err := data.AddRootCertificate(n.db, cert); err != nil {
			return fmt.Errorf("adding CA certificate: %w", err)
		}
	}

	return nil
}

func (n *NativeCertificateProvider) TLSCertificates() ([]tls.Certificate, error) {
	result := []tls.Certificate{}

	keyPairs := []KeyPair{
		n.previousKeypair,
		n.activeKeypair,
	}

	for _, keyPair := range keyPairs {
		cert, err := keyPair.TLSCertificate()
		if err != nil {
			return nil, err
		}

		result = append(result, *cert)
	}

	return result, nil
}

func isAllowedSignatureAlgorithm(alg x509.SignatureAlgorithm) bool {
	for _, a := range allowedSignatureAlgorithms {
		if a == alg {
			return true
		}
	}

	return false
}

func isAllowedPublicKeyAlgorithm(alg x509.PublicKeyAlgorithm) bool {
	for _, a := range allowedPublicKeyAlgorithms {
		if a == alg {
			return true
		}
	}

	return false
}
