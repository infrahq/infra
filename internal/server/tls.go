package server

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"time"

	"github.com/infrahq/secrets"
	"golang.org/x/crypto/acme/autocert"

	"github.com/infrahq/infra/internal/certs"
	"github.com/infrahq/infra/internal/logging"
)

func tlsConfigFromOptions(
	storage map[string]secrets.SecretStorage,
	tlsCacheDir string,
	opts TLSOptions,
) (*tls.Config, error) {
	// TODO: print CA fingerprint when the client can trust that fingerprint

	if opts.Certificate != "" && opts.PrivateKey != "" {
		roots, err := x509.SystemCertPool()
		if err != nil {
			logging.Warnf("failed to load TLS roots from system: %v", err)
			roots = x509.NewCertPool()
		}

		if opts.CA != "" {
			if !roots.AppendCertsFromPEM([]byte(opts.CA)) {
				logging.Warnf("failed to load TLS CA, invalid PEM")
			}
		}

		key, err := secrets.GetSecret(opts.PrivateKey, storage)
		if err != nil {
			return nil, fmt.Errorf("failed to load TLS private key: %w", err)
		}

		cert, err := tls.X509KeyPair([]byte(opts.Certificate), []byte(key))
		if err != nil {
			return nil, fmt.Errorf("failed to load TLS key pair: %w", err)
		}

		return &tls.Config{
			MinVersion: tls.VersionTLS12,
			// enable HTTP/2
			NextProtos:   []string{"h2", "http/1.1"},
			Certificates: []tls.Certificate{cert},
			// enabled optional mTLS
			ClientAuth: tls.VerifyClientCertIfGiven,
			ClientCAs:  roots,
		}, nil
	}

	if err := os.MkdirAll(tlsCacheDir, 0o700); err != nil {
		return nil, fmt.Errorf("create tls cache: %w", err)
	}

	manager := &autocert.Manager{
		Prompt: autocert.AcceptTOS,
		Cache:  autocert.DirCache(tlsCacheDir),
	}
	tlsConfig := manager.TLSConfig()
	tlsConfig.MinVersion = tls.VersionTLS12
	// TODO: enabled optional mTLS when opts.CA is set
	tlsConfig.GetCertificate = SelfSignedOrLetsEncryptCert(manager)

	return tlsConfig, nil
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

			certBytes, keyBytes, err = certs.GenerateCertificate([]string{serverName}, ca, caPrivKey)
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
				Str("Server name", serverName).
				Str("SHA256 fingerprint", certs.Fingerprint(pemDecode(certBytes))).
				Msg("new server certificate")
		}

		keypair, err := tls.X509KeyPair(certBytes, keyBytes)
		if err != nil {
			return nil, err
		}

		return &keypair, nil
	}
}

func pemDecode(raw []byte) []byte {
	block, _ := pem.Decode(raw)
	return block.Bytes
}

// TODO: remove
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
