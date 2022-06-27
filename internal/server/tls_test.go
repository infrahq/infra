package server

import (
	"crypto/tls"
	"crypto/x509"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/infrahq/secrets"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/golden"

	"github.com/infrahq/infra/internal/cmd/types"
)

func TestTLSConfigFromOptions(t *testing.T) {
	storage := map[string]secrets.SecretStorage{
		"plaintext": &secrets.PlainSecretProvider{},
		"file":      &secrets.FileSecretProvider{},
	}

	ca := golden.Get(t, "pki/ca.crt")
	t.Run("user provided certificate", func(t *testing.T) {
		opts := TLSOptions{
			CA:          types.StringOrFile(ca),
			Certificate: types.StringOrFile(golden.Get(t, "pki/localhost.crt")),
			PrivateKey:  "file:testdata/pki/localhost.key",
		}
		config, err := tlsConfigFromOptions(storage, t.TempDir(), opts)
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

		resp, err := client.Get(srv.URL)
		assert.NilError(t, err)
		assert.Equal(t, resp.StatusCode, http.StatusOK)
	})
}

var noopHandler = http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
})
