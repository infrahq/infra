package server

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/pki"
	"github.com/infrahq/infra/uid"
	"gotest.tools/v3/assert"
)

func TestCertificateSigningWorks(t *testing.T) {
	db := setupDB(t)

	cp, err := pki.NewNativeCertificateProvider(db, pki.NativeCertificateProviderConfig{
		FullKeyRotationDurationInDays: 2,
	})
	assert.NilError(t, err)

	err = cp.CreateCA()
	assert.NilError(t, err)

	err = cp.RotateCA()
	assert.NilError(t, err)

	user := &models.Identity{
		Model: models.Model{ID: uid.New()},
		Kind:  models.UserKind,
		Name:  "joe@example.com",
	}

	keyPair, err := pki.MakeUserCert("User "+user.ID.String(), 24*time.Hour)
	assert.NilError(t, err)

	// happens on the server, needs to be a request for this.
	signedCert, signedRaw, err := pki.SignUserCert(cp, keyPair.Cert, user)
	assert.NilError(t, err)

	keyPair.SignedCert = signedCert
	keyPair.SignedCertPEM = signedRaw

	// create a test server and client to make sure the certs work.
	requireMutualTLSWorks(t, keyPair, cp)
}

func requireMutualTLSWorks(t *testing.T, clientKeypair *pki.KeyPair, cp pki.CertificateProvider) {
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "success!")
	}))

	serverTLSCerts, err := cp.TLSCertificates()
	assert.NilError(t, err)

	caPool := x509.NewCertPool()

	for _, cert := range cp.ActiveCAs() {
		cert := cert
		caPool.AddCert(&cert)
	}

	server.TLS = &tls.Config{
		Certificates: serverTLSCerts,
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    caPool,
		MinVersion:   tls.VersionTLS12,
	}

	server.StartTLS()
	defer server.Close()

	// This will response with HTTP 200 OK and a body containing success!. We can now set up the client to trust the CA, and send a request to the server:

	clientTLSCert, err := clientKeypair.TLSCertificate()
	assert.NilError(t, err)

	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			Certificates: []tls.Certificate{*clientTLSCert},
			ClientCAs:    caPool,
			RootCAs:      caPool,
			MinVersion:   tls.VersionTLS12,
		},
	}
	http := http.Client{
		Transport: transport,
	}

	resp, err := http.Get(server.URL)
	assert.NilError(t, err)

	// If no errors occurred, we now have our success! response from the server, and can verify it:

	respBodyBytes, err := ioutil.ReadAll(resp.Body)
	assert.NilError(t, err)

	body := strings.TrimSpace(string(respBodyBytes))
	assert.Equal(t, "success!", body)
}
