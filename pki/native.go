package pki

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"log"
	"math/big"
	"net"
	"os"
	"path"
	"strings"
	"time"
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

	activeKeypair   KeyPair
	previousKeypair KeyPair

	// TODO: support arbitrary storage
	// secretStorage     secrets.SecretStorage
	// secretKeyProvider secrets.SymmetricKeyProvider
}

type NativeCertificateProviderConfig struct {
	StoragePath                   string
	FullKeyRotationDurationInDays int
	// Algorithm string // only ed25519 so far.
}

func NewNativeCertificateProvider(cfg NativeCertificateProviderConfig) (*NativeCertificateProvider, error) {
	if cfg.FullKeyRotationDurationInDays == 0 {
		cfg.FullKeyRotationDurationInDays = 365
	}

	p := &NativeCertificateProvider{
		NativeCertificateProviderConfig: cfg,
	}
	if err := p.loadFromDisk(); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
	}

	return p, nil
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

	n.activeKeypair.CertRaw = pemBytes
	n.activeKeypair.SignedCertRaw = pemBytes
	n.activeKeypair.Cert = cert
	n.activeKeypair.SignedCert = cert

	return n.RotateCA()
}

func (n *NativeCertificateProvider) ActiveCAs() []x509.Certificate {
	result := []x509.Certificate{}

	if n.previousKeypair.Cert != nil && certActive(n.previousKeypair.Cert) {
		result = append(result, *n.previousKeypair.Cert)
	}

	if n.activeKeypair.Cert != nil && certActive(n.activeKeypair.Cert) {
		result = append(result, *n.activeKeypair.Cert)
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
		IPAddresses:        []net.IP{net.ParseIP("127.0.0.1"), net.IPv6loopback},
		SerialNumber:       big.NewInt(2),
		Issuer:             n.activeKeypair.Cert.Subject,
		Subject:            csr.Subject,
		EmailAddresses:     csr.EmailAddresses,
		Extensions:         csr.Extensions,      // TODO: security issue?
		ExtraExtensions:    csr.ExtraExtensions, // TODO: security issue?
		NotBefore:          time.Now(),
		NotAfter:           time.Now().Add(24 * time.Hour),
		KeyUsage:           x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:        []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
	}

	if !n.activeKeypair.Cert.IsCA {
		panic("not ca")
	}

	if n.activeKeypair.Cert.KeyUsage&x509.KeyUsageCertSign != x509.KeyUsageCertSign {
		panic("can't sign keys with this cert")
	}

	signedCert, err := x509.CreateCertificate(randReader, certTemplate, n.activeKeypair.Cert, csr.PublicKey, n.activeKeypair.PrivateKey)
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

	n.activeKeypair.CertRaw = pemBytes
	n.activeKeypair.SignedCertRaw = pemBytes
	n.activeKeypair.Cert = cert
	n.activeKeypair.SignedCert = cert

	return n.saveToDisk()
}

// createCertSignedBy signs the signee public key using the signer private key, allowing anyone to verify the signature with the signer public key. Certificate expires after _lifetime_
func createCertSignedBy(signer, signee KeyPair, lifetime time.Duration) (*x509.Certificate, []byte, error) {
	// sig := ed25519.Sign(signer.PrivateKey, signee.PublicKey)
	// if !ed25519.Verify(signer.PublicKey, signee.PublicKey, sig) {
	// 	return nil, nil, errors.New("self-signed certificate doesn't match signature")
	// }

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
		NotBefore:          time.Now(),
		NotAfter:           time.Now().Add(lifetime),
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
		DNSNames:    []string{"localhost"}, // TODO: Support domain names for services
		IPAddresses: []net.IP{net.IPv4(127, 0, 0, 1), net.IPv6loopback},
		// EmailAddresses []string
		// URIs           []*url.URL
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

func (n *NativeCertificateProvider) saveToDisk() error {
	err := os.MkdirAll(n.StoragePath, 0o600)
	if err != nil && !os.IsExist(err) {
		log.Printf("creating directory %q", n.StoragePath)
	}

	err = writeToFile(path.Join(n.StoragePath, "root.crt"), n.activeKeypair.CertRaw)
	if err != nil {
		return fmt.Errorf("writing PEM: %w", err)
	}

	marshalledPrvKey, err := x509.MarshalPKCS8PrivateKey(n.activeKeypair.PrivateKey)
	if err != nil {
		return fmt.Errorf("marshalling private key: %w", err)
	}

	err = writePEMToFile(path.Join(n.StoragePath, "root.key"), &pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: marshalledPrvKey,
	})
	if err != nil {
		return fmt.Errorf("writing PEM: %w", err)
	}

	err = writeToFile(path.Join(n.StoragePath, "root-previous.crt"), n.previousKeypair.CertRaw)
	if err != nil {
		return fmt.Errorf("writing PEM: %w", err)
	}

	marshalledPrvKey, err = x509.MarshalPKCS8PrivateKey(n.previousKeypair.PrivateKey)
	if err != nil {
		return fmt.Errorf("marshalling private key: %w", err)
	}

	err = writePEMToFile(path.Join(n.StoragePath, "root-previous.key"), &pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: marshalledPrvKey,
	})
	if err != nil {
		return fmt.Errorf("writing PEM: %w", err)
	}

	return nil
}

func writePEMToFile(file string, p *pem.Block) error {
	f, err := os.Create(file)
	if err != nil {
		return fmt.Errorf("creating %s: %w", file, err)
	}

	err = pem.Encode(f, p)
	if err != nil {
		return fmt.Errorf("writing root certificate: %w", err)
	}

	return f.Close()
}

func writeToFile(file string, data []byte) error {
	f, err := os.Create(file)
	if err != nil {
		return fmt.Errorf("creating %s: %w", file, err)
	}

	i, err := f.Write(data)
	if err != nil {
		return fmt.Errorf("writing root certificate: %w", err)
	}

	if i != len(data) {
		return fmt.Errorf("incomplete file write to %s", file)
	}

	return f.Close()
}

func (n *NativeCertificateProvider) loadFromDisk() error {
	var ok bool

	_ = os.MkdirAll(n.StoragePath, 0o700)

	pems, bytes, err := ReadFromPEMFile(path.Join(n.StoragePath, "root.crt"))
	if err != nil {
		return fmt.Errorf("reading cert: %w", err)
	}
	n.activeKeypair.CertRaw = bytes

	cert, err := x509.ParseCertificate(pems[0].Bytes)
	if err != nil {
		return fmt.Errorf("parsing certificate: %w", err)
	}

	n.activeKeypair.Cert = cert

	// nolint:exhaustive
	switch cert.PublicKeyAlgorithm {
	case x509.Ed25519:
		n.activeKeypair.PublicKey, ok = cert.PublicKey.(ed25519.PublicKey)
		if !ok {
			return fmt.Errorf("unexpected key type %t, expected ed25519", cert.PublicKey)
		}
	default:
		panic("unexpected key algorithm " + cert.PublicKeyAlgorithm.String())
	}

	pems, _, err = ReadFromPEMFile(path.Join(n.StoragePath, "root.key"))
	if err != nil {
		return fmt.Errorf("reading PEM: %w", err)
	}

	key, err := x509.ParsePKCS8PrivateKey(pems[0].Bytes)
	if err != nil {
		return fmt.Errorf("decoding key: %w", err)
	}

	n.activeKeypair.PrivateKey = key.(ed25519.PrivateKey)

	pems, bytes, err = ReadFromPEMFile(path.Join(n.StoragePath, "root-previous.crt"))
	if err != nil {
		return fmt.Errorf("reading cert: %w", err)
	}

	n.previousKeypair.CertRaw = bytes

	cert, err = x509.ParseCertificate(pems[0].Bytes)
	if err != nil {
		return fmt.Errorf("parsing certificate: %w", err)
	}

	n.previousKeypair.Cert = cert

	// nolint:exhaustive
	switch cert.PublicKeyAlgorithm {
	case x509.Ed25519:
		n.previousKeypair.PublicKey, ok = cert.PublicKey.(ed25519.PublicKey)
		if !ok {
			return fmt.Errorf("unexpected key type %t, expected ed25519", cert.PublicKey)
		}
	default:
		panic("unexpected key algorithm " + cert.PublicKeyAlgorithm.String())
	}

	pems, _, err = ReadFromPEMFile(path.Join(n.StoragePath, "root-previous.key"))
	if err != nil {
		return fmt.Errorf("reading PEM: %w", err)
	}

	key, err = x509.ParsePKCS8PrivateKey(pems[0].Bytes)
	if err != nil {
		return fmt.Errorf("decoding key: %w", err)
	}

	n.previousKeypair.PrivateKey = key.(ed25519.PrivateKey)

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
