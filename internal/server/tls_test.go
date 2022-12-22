package server

import (
	"crypto/tls"
	"crypto/x509"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/golden"

	"github.com/infrahq/infra/internal/cmd/types"
)

func TestTLSConfigFromOptions(t *testing.T) {
	ca := golden.Get(t, "pki/ca.crt")
	t.Run("user provided certificate", func(t *testing.T) {
		opts := TLSOptions{
			CA:          types.StringOrFile(ca),
			Certificate: types.StringOrFile(golden.Get(t, "pki/localhost.crt")),
			PrivateKey:  types.StringOrFile(golden.Get(t, "pki/localhost.key")),
		}
		config, err := tlsConfigFromOptions(opts)
		assert.NilError(t, err)

		srv := httptest.NewUnstartedServer(noopHandler)
		srv.TLS = config
		srv.StartTLS()
		t.Cleanup(srv.Close)

		roots := x509.NewCertPool()
		roots.AppendCertsFromPEM(ca)
		client := &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{RootCAs: roots, MinVersion: tls.VersionTLS12},
			},
		}

		// nolint:noctx
		resp, err := client.Get(srv.URL)
		assert.NilError(t, err)
		assert.Equal(t, resp.StatusCode, http.StatusOK)
	})

	t.Run("generate TLS cert from CA", func(t *testing.T) {
		if testing.Short() {
			t.Skip("too slow for short run")
		}
		opts := TLSOptions{
			CA:           types.StringOrFile(ca),
			CAPrivateKey: types.StringOrFile(golden.Get(t, "pki/ca.key")),
		}
		config, err := tlsConfigFromOptions(opts)
		assert.NilError(t, err)

		l, err := net.Listen("tcp", "127.0.0.1:0")
		assert.NilError(t, err)

		l = tls.NewListener(l, config)
		// nolint:gosec
		srv := http.Server{Handler: noopHandler}

		go func() {
			// nolint:errorlint
			if err := srv.Serve(l); err != http.ErrServerClosed {
				t.Log(err)
			}
		}()
		t.Cleanup(func() { _ = srv.Close() })

		roots := x509.NewCertPool()
		roots.AppendCertsFromPEM(ca)
		client := &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{RootCAs: roots, MinVersion: tls.VersionTLS12},
			},
		}

		// nolint:noctx
		resp, err := client.Get("https://" + l.Addr().String())
		assert.NilError(t, err)
		assert.Equal(t, resp.StatusCode, http.StatusOK)
	})
}

var noopHandler = http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
})
