package server

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"math/big"
	"net"
	"time"

	"github.com/infrahq/infra/internal/generate"
	"golang.org/x/crypto/acme/autocert"
)

func generateSelfSignedCert(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
	ca := &x509.Certificate{
		SerialNumber: big.NewInt(2019),
		Subject: pkix.Name{
			CommonName:   generate.RandString(25),
			Organization: []string{generate.RandString(25)},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(0, 1, 0),
		IsCA:                  true,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}

	caPrivKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return nil, err
	}

	caBytes, err := x509.CreateCertificate(rand.Reader, ca, ca, &caPrivKey.PublicKey, caPrivKey)
	if err != nil {
		return nil, err
	}

	caPEM := new(bytes.Buffer)
	pem.Encode(caPEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: caBytes,
	})

	caPrivKeyPEM := new(bytes.Buffer)
	pem.Encode(caPrivKeyPEM, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(caPrivKey),
	})

	cert := &x509.Certificate{
		SerialNumber: big.NewInt(1658),
		Subject: pkix.Name{
			CommonName:   generate.RandString(25),
			Organization: []string{generate.RandString(25)},
		},
		IPAddresses:  []net.IP{net.IPv4(127, 0, 0, 1), net.IPv6loopback},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().AddDate(0, 1, 0),
		SubjectKeyId: []byte{1, 2, 3, 4, 6},
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:     x509.KeyUsageDigitalSignature,
	}

	if hello.ServerName != "" {
		cert.DNSNames = []string{hello.ServerName}
	} else {
		tcp, ok := hello.Conn.LocalAddr().(*net.TCPAddr)
		if !ok {
			return nil, errors.New("could not determine ip")
		}
		cert.IPAddresses = append(cert.IPAddresses, tcp.IP)
	}

	certPrivKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return nil, err
	}

	certBytes, err := x509.CreateCertificate(rand.Reader, cert, ca, &certPrivKey.PublicKey, caPrivKey)
	if err != nil {
		return nil, err
	}

	certPEM := new(bytes.Buffer)
	pem.Encode(certPEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certBytes,
	})

	certPrivKeyPEM := new(bytes.Buffer)
	pem.Encode(certPrivKeyPEM, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(certPrivKey),
	})

	keypair, err := tls.X509KeyPair(certPEM.Bytes(), certPrivKeyPEM.Bytes())
	if err != nil {
		return nil, err
	}

	return &keypair, nil
}

func getSelfSignedOrLetsEncryptCert(certManager *autocert.Manager) func(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
	selfSignCache := make(map[string]*tls.Certificate)

	return func(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {

		cert, err := certManager.GetCertificate(hello)
		if err == nil {
			return cert, nil
		}

		name := hello.ServerName
		if name == "" {
			name = hello.Conn.LocalAddr().String()
		}

		cert, ok := selfSignCache[name]
		if !ok {
			cert, err = generateSelfSignedCert(hello)
			if err != nil {
				return nil, err
			}
			selfSignCache[name] = cert
		}

		return cert, nil
	}
}
