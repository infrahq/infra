package certs

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"strings"
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

	keyBytes := pemEncodePrivateKey(x509.MarshalPKCS1PrivateKey(key))
	return PEMEncodeCertificate(certBytes), keyBytes, nil
}

func SelfSignedOrLetsEncryptCert(manager *autocert.Manager) func(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
	return func(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
		ctx := hello.Context()
		cert, err := manager.GetCertificate(hello)
		if err == nil {
			return cert, nil
		}

		serverName := hello.ServerName

		if serverName == "" {
			serverName, _, err = net.SplitHostPort(hello.Conn.LocalAddr().String())
			if err != nil {
				return nil, err
			}
		}

		certBytes, err := manager.Cache.Get(ctx, serverName+".crt")
		if err != nil {
			logging.Warnf("cert: %s", err)
		}

		keyBytes, err := manager.Cache.Get(ctx, serverName+".key")
		if err != nil {
			logging.Warnf("key: %s", err)
		}

		// if either cert or key is missing, create it
		if certBytes == nil || keyBytes == nil {
			ca, caPrivKey, err := newCA()
			if err != nil {
				return nil, err
			}

			certBytes, keyBytes, err = GenerateCertificate([]string{serverName}, ca, caPrivKey)
			if err != nil {
				return nil, err
			}

			if err := manager.Cache.Put(ctx, serverName+".crt", certBytes); err != nil {
				return nil, err
			}

			if err := manager.Cache.Put(ctx, serverName+".key", keyBytes); err != nil {
				return nil, err
			}

			logging.L.Info().
				Str("serverName", serverName).
				Str("fingerprint", Fingerprint(pemDecode(certBytes))).
				Msg("new server certificate")
		}

		keypair, err := tls.X509KeyPair(certBytes, keyBytes)
		if err != nil {
			return nil, err
		}

		return &keypair, nil
	}
}

// Fingerprint returns a sha256 checksum of the certificate formatted as
// hex pairs separated by colons. This is a common format used by browsers.
// The bytes must be the ASN.1 DER form of the x509.Certificate.
func Fingerprint(raw []byte) string {
	checksum := sha256.Sum256(raw)
	s := strings.ReplaceAll(fmt.Sprintf("% x", checksum), " ", ":")
	return strings.ToUpper(s)
}

func pemDecode(raw []byte) []byte {
	block, _ := pem.Decode(raw)
	return block.Bytes
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

func newCA() (*x509.Certificate, *rsa.PrivateKey, error) {
	// Generate a CA to sign self-signed certificates
	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, nil, err
	}

	caPrivKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return nil, nil, err
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
		IsCA:                  true,
		BasicConstraintsValid: true,
	}

	caBytes, err := x509.CreateCertificate(rand.Reader, ca, ca, &caPrivKey.PublicKey, caPrivKey)
	if err != nil {
		return nil, nil, err
	}

	// TODO: is there really no other way to get the Raw field populated?
	ca, _ = x509.ParseCertificate(caBytes)

	return ca, caPrivKey, nil
}
