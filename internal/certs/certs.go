package certs

import (
	"bytes"
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"time"

	"golang.org/x/crypto/acme/autocert"

	"github.com/infrahq/infra/internal/logging"
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

	certPEMBuf := new(bytes.Buffer)
	if err := pem.Encode(certPEMBuf, &pem.Block{Type: "CERTIFICATE", Bytes: certBytes}); err != nil {
		return nil, nil, err
	}

	keyPEMBuf := new(bytes.Buffer)
	if err := pem.Encode(keyPEMBuf, &pem.Block{Type: "PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)}); err != nil {
		return nil, nil, err
	}

	return certPEMBuf.Bytes(), keyPEMBuf.Bytes(), nil
}

func SelfSignedOrLetsEncryptCert(manager *autocert.Manager, serverName string) func(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
	return func(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
		cert, err := manager.GetCertificate(hello)
		if err == nil {
			return cert, nil
		}

		if serverName == "" {
			serverName = hello.ServerName
		}

		if serverName == "" {
			serverName = hello.Conn.LocalAddr().String()
		}

		certBytes, err := manager.Cache.Get(context.TODO(), serverName+".crt")
		if err != nil {
			logging.S.Warnf("cert: %s", err)
		}

		keyBytes, err := manager.Cache.Get(context.TODO(), serverName+".key")
		if err != nil {
			logging.S.Warnf("key: %s", err)
		}

		// if either cert or key is missing, create it
		if certBytes == nil || keyBytes == nil {
			// Generate a CA to sign self-signed certificates
			serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
			if err != nil {
				return nil, err
			}

			ca := &x509.Certificate{
				SerialNumber: serialNumber,
				Subject: pkix.Name{
					Organization: []string{"Infra"},
				},
				NotBefore:             time.Now().Add(-5 * time.Minute).UTC(),
				NotAfter:              time.Now().AddDate(0, 0, 365).UTC(),
				KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment | x509.KeyUsageCertSign,
				ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
				BasicConstraintsValid: true,
			}

			caPrivKey, err := rsa.GenerateKey(rand.Reader, 4096)
			if err != nil {
				return nil, err
			}

			certBytes, keyBytes, err = GenerateCertificate([]string{serverName}, ca, caPrivKey)
			if err != nil {
				return nil, err
			}

			if err := manager.Cache.Put(context.TODO(), serverName+".crt", certBytes); err != nil {
				return nil, err
			}

			if err := manager.Cache.Put(context.TODO(), serverName+".key", keyBytes); err != nil {
				return nil, err
			}
		}

		keypair, err := tls.X509KeyPair(certBytes, keyBytes)
		if err != nil {
			return nil, err
		}

		return &keypair, nil
	}
}
