package server

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net"
	"sync"

	"golang.org/x/crypto/acme/autocert"

	"github.com/infrahq/infra/internal/certs"
	"github.com/infrahq/infra/internal/logging"
)

// MapCache is a simple in-memory caching mechanism, it is not thread safe
type MapCache map[string][]byte

func (m MapCache) Get(_ context.Context, name string) ([]byte, error) {
	data, cached := m[name]
	if !cached {
		return nil, autocert.ErrCacheMiss
	}
	return data, nil
}

func (m MapCache) Put(_ context.Context, name string, data []byte) error {
	m[name] = data
	return nil
}

func (m MapCache) Delete(_ context.Context, name string) error {
	delete(m, name)
	return nil
}

func tlsConfigFromOptions(opts TLSOptions) (*tls.Config, error) {
	// TODO: how can we test this?
	if opts.ACME {
		manager := &autocert.Manager{
			Prompt: autocert.AcceptTOS,
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
		if len(raw) == 0 {
			logging.Errorf("could not read CA %q", opts.CA)
		} else {
			logging.L.Info().
				Str("SHA256 fingerprint", certs.Fingerprint(raw)).
				Msg("TLS CA")

			if !roots.AppendCertsFromPEM([]byte(opts.CA)) {
				logging.Warnf("failed to load TLS CA, invalid PEM")
			}
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
		cert, err := tls.X509KeyPair([]byte(opts.Certificate), []byte(opts.PrivateKey))
		if err != nil {
			return nil, fmt.Errorf("failed to load TLS key pair: %w", err)
		}

		cfg.Certificates = []tls.Certificate{cert}
		return cfg, nil
	}

	if opts.CA == "" || opts.CAPrivateKey == "" {
		return nil, fmt.Errorf("either a TLS certificate and key or a TLS CA and key is required")
	}

	ca := keyPair{cert: []byte(opts.CA), key: []byte(opts.CAPrivateKey)}
	certCache := make(MapCache)

	cfg.GetCertificate = getCertificate(certCache, ca)
	return cfg, nil
}

type keyPair struct {
	cert []byte
	key  []byte
}

func getCertificate(cache autocert.Cache, ca keyPair) func(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
	var lock sync.RWMutex

	getKeyPair := func(serverName string) (cert, key []byte) {
		certBytes, _ := cache.Get(context.TODO(), serverName+".crt")
		keyBytes, _ := cache.Get(context.TODO(), serverName+".key")
		if certBytes == nil || keyBytes == nil {
			logging.Infof("no cached TLS cert for %v", serverName)
		}
		return certBytes, keyBytes
	}

	return func(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
		serverName := hello.ServerName

		if serverName == "" {
			var err error
			serverName, _, err = net.SplitHostPort(hello.Conn.LocalAddr().String())
			if err != nil {
				return nil, err
			}
		}

		lock.RLock()
		certBytes, keyBytes := getKeyPair(serverName)
		lock.RUnlock()
		if certBytes != nil && keyBytes != nil {
			return tlsCertFromKeyPair(certBytes, keyBytes)
		}

		lock.Lock()
		// must check again after write lock is acquired
		certBytes, keyBytes = getKeyPair(serverName)
		defer lock.Unlock()
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

		if err := cache.Put(context.TODO(), serverName+".crt", certBytes); err != nil {
			return nil, err
		}

		if err := cache.Put(context.TODO(), serverName+".key", keyBytes); err != nil {
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
	if block != nil {
		return block.Bytes
	}
	return []byte{}
}

func tlsCertFromKeyPair(cert, key []byte) (*tls.Certificate, error) {
	keypair, err := tls.X509KeyPair(cert, key)
	if err != nil {
		return nil, err
	}
	return &keypair, nil
}
