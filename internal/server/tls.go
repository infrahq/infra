package server

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net"
	"os"
	"sync"

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
	// TODO: how can we test this?
	if opts.ACME {
		if err := os.MkdirAll(tlsCacheDir, 0o700); err != nil {
			return nil, fmt.Errorf("create tls cache: %w", err)
		}

		manager := &autocert.Manager{
			Prompt: autocert.AcceptTOS,
			Cache:  autocert.DirCache(tlsCacheDir),
			// TODO: according to the docs HostPolicy should be set to prevent
			// a DoS attack on certificate requests.
			// See https://github.com/infrahq/infra/issues/2484
		}
		tlsConfig := manager.TLSConfig()
		tlsConfig.MinVersion = tls.VersionTLS12
		return tlsConfig, nil
	}

	// TODO: print CA fingerprint when the client can trust that fingerprint

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

	cfg := &tls.Config{
		MinVersion: tls.VersionTLS12,
		// enable HTTP/2
		NextProtos: []string{"h2", "http/1.1"},
		// enabled optional mTLS
		ClientAuth: tls.VerifyClientCertIfGiven,
		ClientCAs:  roots,
	}

	if opts.Certificate != "" && opts.PrivateKey != "" {
		key, err := secrets.GetSecret(opts.PrivateKey, storage)
		if err != nil {
			return nil, fmt.Errorf("failed to load TLS private key: %w", err)
		}

		cert, err := tls.X509KeyPair([]byte(opts.Certificate), []byte(key))
		if err != nil {
			return nil, fmt.Errorf("failed to load TLS key pair: %w", err)
		}

		cfg.Certificates = []tls.Certificate{cert}
		return cfg, nil
	}

	if opts.CA == "" || opts.CAPrivateKey == "" {
		return nil, fmt.Errorf("either a TLS certificate and key or a TLS CA and key is required")
	}

	key, err := secrets.GetSecret(opts.CAPrivateKey, storage)
	if err != nil {
		return nil, fmt.Errorf("failed to load TLS CA private key: %w", err)
	}

	ca := keyPair{cert: []byte(opts.CA), key: []byte(key)}
	cfg.GetCertificate = getCertificate(autocert.DirCache(tlsCacheDir), ca)
	return cfg, nil
}

type keyPair struct {
	cert []byte
	key  []byte
}

func getCertificate(cache autocert.Cache, ca keyPair) func(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
	var lock sync.RWMutex

	getKeyPair := func(ctx context.Context, serverName string) (cert, key []byte) {
		certBytes, _ := cache.Get(ctx, serverName+".crt")
		keyBytes, _ := cache.Get(ctx, serverName+".key")
		if certBytes == nil || keyBytes == nil {
			logging.Infof("no cached TLS cert for %v", serverName)
		}
		return certBytes, keyBytes
	}

	return func(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
		ctx := hello.Context()
		serverName := hello.ServerName

		if serverName == "" {
			var err error
			serverName, _, err = net.SplitHostPort(hello.Conn.LocalAddr().String())
			if err != nil {
				return nil, err
			}
		}

		lock.RLock()
		certBytes, keyBytes := getKeyPair(ctx, serverName)
		lock.RUnlock()
		if certBytes != nil && keyBytes != nil {
			return tlsCertFromKeyPair(certBytes, keyBytes)
		}

		lock.Lock()
		// must check again after write lock is acquired
		certBytes, keyBytes = getKeyPair(ctx, serverName)
		defer lock.Unlock()
		if certBytes != nil && keyBytes != nil {
			return tlsCertFromKeyPair(certBytes, keyBytes)
		}

		// if either cert or key is missing, create it
		ca, err := tls.X509KeyPair(ca.cert, ca.key)
		if err != nil {
			return nil, err
		}

		caCert, err := x509.ParseCertificate(ca.Certificate[0])
		if err != nil {
			return nil, err
		}

		hosts := []string{"127.0.0.1", "::1", serverName}
		certBytes, keyBytes, err = certs.GenerateCertificate(hosts, caCert, ca.PrivateKey)
		if err != nil {
			return nil, err
		}

		if err := cache.Put(ctx, serverName+".crt", certBytes); err != nil {
			return nil, err
		}

		if err := cache.Put(ctx, serverName+".key", keyBytes); err != nil {
			return nil, err
		}

		logging.L.Info().
			Str("Server name", serverName).
			Str("SHA256 fingerprint", certs.Fingerprint(pemDecode(certBytes))).
			Msg("new server certificate")

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

func tlsCertFromKeyPair(cert, key []byte) (*tls.Certificate, error) {
	keypair, err := tls.X509KeyPair(cert, key)
	if err != nil {
		return nil, err
	}
	return &keypair, nil
}
