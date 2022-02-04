package pki

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"math/big"
	"os"
	"path"
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

	activeKeypair   keyPair
	previousKeypair keyPair

	// TODO: support arbitrary storage
	// secretStorage     secrets.SecretStorage
	// secretKeyProvider secrets.SymmetricKeyProvider
}

type NativeCertificateProviderConfig struct {
	StoragePath                   string
	FullKeyRotationDurationInDays int
	// Algorithm string // only ed25519 so far.
}

type keyPair struct {
	PublicKey  ed25519.PublicKey
	PrivateKey ed25519.PrivateKey
	certRaw    []byte
	Cert       *x509.Certificate
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

	n.activeKeypair.certRaw = raw
	n.activeKeypair.Cert = cert

	return n.RotateCA()
}

func (n *NativeCertificateProvider) ActiveCAs() []x509.Certificate {
	result := []x509.Certificate{}

	if certActive(n.previousKeypair.Cert) {
		result = append(result, *n.previousKeypair.Cert)
	}

	if certActive(n.activeKeypair.Cert) {
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

func (n *NativeCertificateProvider) SignCertificate(csr x509.CertificateRequest) (pemBytes []byte, err error) {
	switch csr.Subject.CommonName {
	case rootCAName:
		return nil, fmt.Errorf("cannot sign cert pretending to be the root CA")
	case "Engine", "Server", "Client":
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

		SerialNumber: big.NewInt(2),
		Issuer:       n.activeKeypair.Cert.Subject,
		Subject:      csr.Subject,
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}

	rawCert, err := x509.CreateCertificate(randReader, certTemplate, n.activeKeypair.Cert, csr.PublicKey, n.activeKeypair.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("creating cert: %w", err)
	}

	pemBytes = pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: rawCert,
	})

	return pemBytes, nil
}

// RotateCA does a half-rotation. the current cert becomes the previous cert, and there are always two active certificates
func (n *NativeCertificateProvider) RotateCA() error {
	n.previousKeypair = n.activeKeypair

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

	n.activeKeypair.certRaw = raw
	n.activeKeypair.Cert = cert

	return n.saveToDisk()
}

// createCertSignedBy signs the signee public key using the signer private key, allowing anyone to verify the signature with the signer public key. Certificate expires after _lifetime_
func createCertSignedBy(signer, signee keyPair, lifetime time.Duration) (*x509.Certificate, []byte, error) {
	sig := ed25519.Sign(signer.PrivateKey, signee.PublicKey)
	if !ed25519.Verify(signer.PublicKey, signee.PublicKey, sig) {
		return nil, nil, errors.New("self-signed certificate doesn't match signature")
	}

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)

	serial, err := rand.Int(randReader, serialNumberLimit)
	if err != nil {
		return nil, nil, fmt.Errorf("creating random serial: %w", err)
	}

	certTemplate := x509.Certificate{
		Signature:          sig,
		SignatureAlgorithm: x509.PureEd25519,
		PublicKeyAlgorithm: x509.Ed25519,
		PublicKey:          signee.PublicKey,
		SerialNumber:       serial,
		Issuer:             pkix.Name{CommonName: rootCAName},
		Subject:            pkix.Name{CommonName: rootCAName},
		NotBefore:          time.Now(),
		NotAfter:           time.Now().Add(lifetime),
		KeyUsage:           x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:        []x509.ExtKeyUsage{},
		IsCA:               true,
	}

	// create client certificate from template and CA public key
	rawCert, err := x509.CreateCertificate(rand.Reader, &certTemplate, &certTemplate, signee.PublicKey, signee.PrivateKey)
	if err != nil {
		return nil, nil, fmt.Errorf("creating certificate: %w", err)
	}

	cert, err := x509.ParseCertificate(rawCert)
	if err != nil {
		return nil, nil, fmt.Errorf("parsing self-created certificate: %w", err)
	}

	cert.IsCA = true // this isn't persisted

	return cert, rawCert, nil
}

func (n *NativeCertificateProvider) saveToDisk() error {
	err := os.MkdirAll(n.StoragePath, 0o600)
	if err != nil && !os.IsExist(err) {
		log.Printf("creating directory %q", n.StoragePath)
	}

	err = writePEMToFile(path.Join(n.StoragePath, "root.crt"), &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: n.activeKeypair.certRaw,
	})
	if err != nil {
		return fmt.Errorf("writing PEM: %w", err)
	}

	err = writePEMToFile(path.Join(n.StoragePath, "root.key"), &pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: []byte(base64.StdEncoding.EncodeToString(n.activeKeypair.PrivateKey)),
	})
	if err != nil {
		return fmt.Errorf("writing PEM: %w", err)
	}

	err = writePEMToFile(path.Join(n.StoragePath, "root-previous.crt"), &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: n.previousKeypair.certRaw,
	})
	if err != nil {
		return fmt.Errorf("writing PEM: %w", err)
	}

	err = writePEMToFile(path.Join(n.StoragePath, "root-previous.key"), &pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: []byte(base64.StdEncoding.EncodeToString(n.activeKeypair.PrivateKey)),
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

func (n *NativeCertificateProvider) loadFromDisk() error {
	var ok bool

	err := os.MkdirAll(n.StoragePath, 0o600)
	if err != nil && !os.IsExist(err) {
		log.Printf("creating directory %q", n.StoragePath)
	}

	pems, err := readFromPEMFile(path.Join(n.StoragePath, "root.crt"))
	if err != nil {
		return fmt.Errorf("reading PEM: %w", err)
	}

	n.activeKeypair.certRaw = pems[0].Bytes

	cert, err := x509.ParseCertificate(n.activeKeypair.certRaw)
	if err != nil {
		return fmt.Errorf("parsing certificate: %w", err)
	}

	cert.IsCA = true
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

	pems, err = readFromPEMFile(path.Join(n.StoragePath, "root.key"))
	if err != nil {
		return fmt.Errorf("reading PEM: %w", err)
	}

	b, err := base64.StdEncoding.DecodeString(string(pems[0].Bytes))
	if err != nil {
		return fmt.Errorf("decoding key: %w", err)
	}

	n.activeKeypair.PrivateKey = b

	pems, err = readFromPEMFile(path.Join(n.StoragePath, "root-previous.crt"))
	if err != nil {
		return fmt.Errorf("reading PEM: %w", err)
	}

	n.previousKeypair.certRaw = pems[0].Bytes

	cert, err = x509.ParseCertificate(n.previousKeypair.certRaw)
	if err != nil {
		return fmt.Errorf("parsing certificate: %w", err)
	}

	cert.IsCA = true
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

	pems, err = readFromPEMFile(path.Join(n.StoragePath, "root-previous.key"))
	if err != nil {
		return fmt.Errorf("reading PEM: %w", err)
	}

	b, err = base64.StdEncoding.DecodeString(string(pems[0].Bytes))
	if err != nil {
		return fmt.Errorf("decoding key: %w", err)
	}

	n.previousKeypair.PrivateKey = b

	return nil
}

func (n *NativeCertificateProvider) TLSCertificates() ([]tls.Certificate, error) {
	result := []tls.Certificate{}

	keyPairs := []keyPair{
		n.previousKeypair,
		n.activeKeypair,
	}
	for _, keyPair := range keyPairs {
		certPEM := pem.EncodeToMemory(&pem.Block{
			Type:  "CERTIFICATE",
			Bytes: keyPair.certRaw,
		})

		keyPEM := pem.EncodeToMemory(&pem.Block{
			Type:  "PRIVATE KEY",
			Bytes: keyPair.PrivateKey,
		})

		cert, err := tls.X509KeyPair(certPEM, keyPEM)
		if err != nil {
			return nil, err
		}

		result = append(result, cert)
	}

	return result, nil
}

func readFromPEMFile(file string) (pems []*pem.Block, err error) {
	// nicer errors from os.Stat. it'll be an errors.Is(err, os.ErrNotExist) if it doesn't exist.
	if _, err = os.Stat(file); err != nil {
		return nil, err
	}

	f, err := os.Open(file)
	if err != nil {
		return nil, fmt.Errorf("opening %q: %w", file, err)
	}
	defer f.Close()

	b, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf("reading %q: %w", file, err)
	}

	for {
		block, rest := pem.Decode(b)
		if block == nil && bytes.Equal(rest, b) {
			return nil, fmt.Errorf("%q contains no pem data", file)
		}

		if block != nil {
			pems = append(pems, block)
		}

		if len(rest) == 0 {
			break
		}
	}

	return pems, nil
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
