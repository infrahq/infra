package certs

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"strings"
	"time"
)

func GenerateCertificate(hosts []string, caCert *x509.Certificate, caKey crypto.PrivateKey) (certPEM []byte, keyPEM []byte, err error) {
	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, nil, err
	}

	cert := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"Infra"},
		},
		NotBefore:             time.Now().Add(-5 * time.Minute).UTC(),
		NotAfter:              time.Now().AddDate(0, 0, 365).UTC(),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	for _, h := range hosts {
		if ip := net.ParseIP(h); ip != nil {
			cert.IPAddresses = append(cert.IPAddresses, ip)
		} else {
			cert.DNSNames = append(cert.DNSNames, h)
		}
	}

	// Create the private key
	key, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return nil, nil, err
	}

	// Create the public certificate signed by the CA
	certBytes, err := x509.CreateCertificate(rand.Reader, &cert, caCert, &key.PublicKey, caKey)
	if err != nil {
		return nil, nil, err
	}

	keyBytes := pemEncodePrivateKey(x509.MarshalPKCS1PrivateKey(key))
	return PEMEncodeCertificate(certBytes), keyBytes, nil
}

// Fingerprint returns a sha256 checksum of the certificate formatted as
// hex pairs separated by colons. This is a common format used by browsers.
// The bytes must be the ASN.1 DER form of the x509.Certificate.
func Fingerprint(raw []byte) string {
	checksum := sha256.Sum256(raw)
	s := strings.ReplaceAll(fmt.Sprintf("% x", checksum), " ", ":")
	return strings.ToUpper(s)
}

// PEMEncodeCertificate accepts the bytes of a x509 certificate in ASN.1 DER form
// and returns a PEM encoded representation of that certificate.
func PEMEncodeCertificate(raw []byte) []byte {
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: raw})
}

// pemEncodePrivateKey accepts the ASN.1 DER form of a private key and returns the
// PEM encoded representation of that private key.
func pemEncodePrivateKey(raw []byte) []byte {
	return pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: raw})
}
