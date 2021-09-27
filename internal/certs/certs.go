package certs

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"time"

	"golang.org/x/crypto/acme/autocert"
)

func SelfSignedCert(hosts []string) ([]byte, []byte, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, err
	}

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)

	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, nil, err
	}

	cert := x509.Certificate{
		PublicKeyAlgorithm: x509.Ed25519,
		SignatureAlgorithm: x509.ECDSAWithSHA512,
		SerialNumber:       serialNumber,
		Subject: pkix.Name{
			Organization: []string{"Infra"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(365, 0, 0),
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment | x509.KeyUsageCertSign,
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

	certBytes, err := x509.CreateCertificate(rand.Reader, &cert, &cert, pub, priv)
	if err != nil {
		return nil, nil, err
	}

	certPEM := new(bytes.Buffer)
	if err := pem.Encode(certPEM, &pem.Block{Type: "CERTIFICATE", Bytes: certBytes}); err != nil {
		return nil, nil, err
	}

	keyBytes, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		return nil, nil, err
	}

	keyPEM := new(bytes.Buffer)
	if err := pem.Encode(keyPEM, &pem.Block{Type: "PRIVATE KEY", Bytes: keyBytes}); err != nil {
		return nil, nil, err
	}

	return certPEM.Bytes(), keyPEM.Bytes(), nil
}

func SelfSignedOrLetsEncryptCert(manager *autocert.Manager, serverNameFunc func() string) func(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
	return func(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
		cert, err := manager.GetCertificate(hello)
		if err == nil {
			return cert, nil
		}

		serverName := serverNameFunc()
		if serverName == "" {
			serverName = hello.ServerName
		}

		name := serverName
		if name == "" {
			name = hello.Conn.LocalAddr().String()
		}

		certBytes, keyBytes, err := func() ([]byte, []byte, error) {
			certBytes, err := manager.Cache.Get(context.TODO(), name+".crt")
			if err != nil {
				return nil, nil, err
			}

			keyBytes, err := manager.Cache.Get(context.TODO(), name+".key")
			if err != nil {
				return nil, nil, err
			}

			return certBytes, keyBytes, nil
		}()
		if err != nil {
			certBytes, keyBytes, err = SelfSignedCert([]string{name})
			if err != nil {
				return nil, err
			}

			if err := manager.Cache.Put(context.TODO(), name+".crt", certBytes); err != nil {
				return nil, err
			}

			if err := manager.Cache.Put(context.TODO(), name+".key", keyBytes); err != nil {
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
