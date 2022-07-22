package server

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net"
	"sync"

	"github.com/infrahq/secrets"
	"golang.org/x/crypto/acme/autocert"

	"github.com/infrahq/infra/internal/certs"
	"github.com/infrahq/infra/internal/logging"
)

// MapCache is a simple in-memory caching mechanism, it is not thread safe
type MapCache map[string][]byte

func (m MapCache) Get(ctx context.Context, name string) ([]byte, error) {
	data, cached := m[name]
	if !cached {
		return nil, autocert.ErrCacheMiss
	}
	return data, nil
}

func (m MapCache) Put(ctx context.Context, name string, data []byte) error {
	m[name] = data
	return nil
}

func (m MapCache) Delete(ctx context.Context, name string) error {
	delete(m, name)
	return nil
}

func (m MapCache) getKeyPair(ctx context.Context, serverName string) (cert, key []byte) {
	certBytes, _ := m.Get(ctx, serverName+".crt")
	keyBytes, _ := m.Get(ctx, serverName+".key")
	if certBytes == nil || keyBytes == nil {
		logging.Infof("no cached TLS cert for %v", serverName)
	}
	return certBytes, keyBytes
}

var certCache = make(MapCache)

var ca keyPair

var cacheLock sync.RWMutex

func tlsConfigFromOptions(
	storage map[string]secrets.SecretStorage,
	opts TLSOptions,
) (*tls.Config, error) {
	// TODO: how can we test this?
	if opts.ACME {
		manager := &autocert.Manager{
			Prompt: autocert.AcceptTOS,
			// Cache:  autocert.DirCache(tlsCacheDir),
			// TODO: according to the docs HostPolicy should be set to prevent
			// a DoS attack on certificate requests.
			// See https://github.com/infrahq/infra/issues/2484
		}
		tlsConfig := manager.TLSConfig()
		tlsConfig.MinVersion = tls.VersionTLS12
		return tlsConfig, nil
	}

	roots, err := x509.SystemCertPool()
	if err != nil {
		logging.L.Err(err).Msgf("failed to load TLS roots from system")
		roots = x509.NewCertPool()
	}

	if opts.CA != "" {
		raw := pemDecode([]byte(opts.CA))
		logging.L.Info().
			Str("SHA256 fingerprint", certs.Fingerprint(raw)).
			Msg("TLS CA")

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

	ca = keyPair{cert: []byte(opts.CA), key: []byte(key)}
	cfg.GetCertificate = getCertificate
	return cfg, nil
}

type keyPair struct {
	cert []byte
	key  []byte
}

func getCertificate(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
	ctx := hello.Context()
	serverName := hello.ServerName

	if serverName == "" {
		var err error
		serverName, _, err = net.SplitHostPort(hello.Conn.LocalAddr().String())
		if err != nil {
			return nil, err
		}
	}

	cacheLock.RLock()
	certBytes, keyBytes := certCache.getKeyPair(ctx, serverName)
	cacheLock.RUnlock()
	if certBytes != nil && keyBytes != nil {
		return tlsCertFromKeyPair(certBytes, keyBytes)
	}

	cacheLock.Lock()
	// must check again after write lock is acquired
	certBytes, keyBytes = certCache.getKeyPair(ctx, serverName)
	defer cacheLock.Unlock()
	if certBytes != nil && keyBytes != nil {
		return tlsCertFromKeyPair(certBytes, keyBytes)
	}

	// if either cert or key is missing, create it
	caTLSCert, err := tls.X509KeyPair(ca.cert, ca.key)
	if err != nil {
		return nil, err
	}

	caCert, err := x509.ParseCertificate(caTLSCert.Certificate[0])
	if err != nil {
		return nil, err
	}

	hosts := []string{"127.0.0.1", "::1", serverName}
	certBytes, keyBytes, err = certs.GenerateCertificate(hosts, caCert, caTLSCert.PrivateKey)
	if err != nil {
		return nil, err
	}

	// append the CA PEM to the cert PEM so that the full chain is available
	// to clients. Not strictly required by TLS, but we do this so that the
	// CLI can prompt the user to trust the CA.
	certBytes = append(certBytes, ca.cert...)

	if err := certCache.Put(ctx, serverName+".crt", certBytes); err != nil {
		return nil, err
	}

	if err := certCache.Put(ctx, serverName+".key", keyBytes); err != nil {
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
