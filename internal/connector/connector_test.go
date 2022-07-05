package connector

import (
	"bytes"
	"crypto"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"gopkg.in/square/go-jose.v2"
	"gopkg.in/square/go-jose.v2/jwt"
	"gotest.tools/v3/assert"

	"github.com/infrahq/infra/internal/claims"
	"github.com/infrahq/infra/internal/server"
)

func TestAuthenticator_Authenticate(t *testing.T) {
	type testCase struct {
		name        string
		setup       func(t *testing.T, req *http.Request)
		fakeClient  fakeClient
		expectedErr string
		expected    func(t *testing.T, claims claims.Custom)
	}

	pub, priv := generateJWK(t)

	run := func(t *testing.T, tc testCase) {
		req := httptest.NewRequest(http.MethodGet, "/apis", nil)
		if tc.setup != nil {
			tc.setup(t, req)
		}

		authn := newAuthenticator("https://127.0.0.1:12345", Options{SkipTLSVerify: true})
		authn.client = tc.fakeClient

		actual, err := authn.Authenticate(req)
		if tc.expectedErr != "" {
			assert.ErrorContains(t, err, tc.expectedErr)
			return
		}
		assert.NilError(t, err)

		if tc.expected != nil {
			tc.expected(t, actual)
		}
	}

	testCases := []testCase{
		{
			name:        "no auth header",
			expectedErr: "no bearer token found",
		},
		{
			name: "no token",
			setup: func(t *testing.T, req *http.Request) {
				req.Header.Set("Authorization", "username:password")
			},
			expectedErr: "invalid JWT signature",
		},
		{
			name: "invalid JWK",
			setup: func(t *testing.T, req *http.Request) {
				req.Header.Set("Authorization", "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwiZW1haWwiOiJ0ZXN0QHRlc3QuY29tIiwiaWF0IjoxNTE2MjM5MDIyfQ.j7o5o8GBkybaYXdFJIi8O6mPF50E-gJWZ3reLfMQD68")
			},
			fakeClient:  fakeClient{err: fmt.Errorf("server not available")},
			expectedErr: "get JWK from server: server not available",
		},
		{
			name: "invalid JWT",
			setup: func(t *testing.T, req *http.Request) {
				req.Header.Set("Authorization", "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwiZW1haWwiOiJ0ZXN0QHRlc3QuY29tIiwiaWF0IjoxNTE2MjM5MDIyfQ.j7o5o8GBkybaYXdFJIi8O6mPF50E-gJWZ3reLfMQD68")
			},
			fakeClient:  fakeClient{key: *pub},
			expectedErr: "error in cryptographic primitive",
		},
		{
			name: "expired JWT",
			setup: func(t *testing.T, req *http.Request) {
				j := generateJWT(t, priv, "test@example.com", time.Now().Add(-1*time.Hour))
				req.Header.Set("Authorization", "Bearer "+j)
			},
			fakeClient:  fakeClient{key: *pub},
			expectedErr: "token is expired",
		},
		{
			name: "no username in JWT",
			setup: func(t *testing.T, req *http.Request) {
				j := generateJWT(t, priv, "", time.Now().Add(time.Hour))
				req.Header.Set("Authorization", "Bearer "+j)
			},
			fakeClient:  fakeClient{key: *pub},
			expectedErr: "no username in JWT claim",
		},
		{
			name: "valid JWT",
			setup: func(t *testing.T, req *http.Request) {
				j := generateJWT(t, priv, "test@example.com", time.Now().Add(time.Hour))
				req.Header.Set("Authorization", "Bearer "+j)
			},
			fakeClient: fakeClient{key: *pub},
			expected: func(t *testing.T, actual claims.Custom) {
				expected := claims.Custom{
					Name:   "test@example.com",
					Groups: []string{"developers"},
				}
				assert.DeepEqual(t, actual, expected)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			run(t, tc)
		})
	}
}

func generateJWK(t *testing.T) (pub *jose.JSONWebKey, priv *jose.JSONWebKey) {
	t.Helper()
	pubkey, key, err := ed25519.GenerateKey(rand.Reader)
	assert.NilError(t, err)

	priv = &jose.JSONWebKey{Key: key, KeyID: "", Algorithm: string(jose.ED25519), Use: "sig"}
	thumb, err := priv.Thumbprint(crypto.SHA256)
	assert.NilError(t, err)

	kid := base64.URLEncoding.EncodeToString(thumb)
	priv.KeyID = kid
	pub = &jose.JSONWebKey{Key: pubkey, KeyID: kid, Algorithm: string(jose.ED25519), Use: "sig"}
	return pub, priv
}

func generateJWT(t *testing.T, priv *jose.JSONWebKey, email string, expiry time.Time) string {
	t.Helper()
	signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.EdDSA, Key: priv}, (&jose.SignerOptions{}).WithType("JWT"))
	assert.NilError(t, err)

	cl := jwt.Claims{
		Issuer:   "InfraHQ",
		Expiry:   jwt.NewNumericDate(expiry),
		IssuedAt: jwt.NewNumericDate(time.Now()),
	}

	custom := claims.Custom{
		Name:   email,
		Groups: []string{"developers"},
	}

	raw, err := jwt.Signed(signer).Claims(cl).Claims(custom).CompactSerialize()
	assert.NilError(t, err)
	return raw
}

type fakeClient struct {
	key jose.JSONWebKey
	err error
}

func (f fakeClient) Do(_ *http.Request) (*http.Response, error) {
	if f.err != nil {
		return nil, f.err
	}

	r := server.WellKnownJWKResponse{Keys: []jose.JSONWebKey{f.key}}

	var buf bytes.Buffer
	err := json.NewEncoder(&buf).Encode(r)
	if err != nil {
		return nil, err
	}

	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       ioutil.NopCloser(&buf),
	}
	return resp, nil
}

func TestCertCache_Certificate(t *testing.T) {
	if testing.Short() {
		t.Skip("too slow for short run")
	}
	testCACertPEM, err := os.ReadFile("./_testdata/test-ca-cert.pem")
	assert.NilError(t, err)

	testCAKeyPEM, err := os.ReadFile("./_testdata/test-ca-key.pem")
	assert.NilError(t, err)

	t.Run("no cached certificate adds empty certificate", func(t *testing.T) {
		certCache := NewCertCache(testCACertPEM, testCAKeyPEM)

		cert, err := certCache.Certificate()

		assert.NilError(t, err)
		assert.Equal(t, len(certCache.hosts), 1)
		assert.Equal(t, certCache.hosts[0], "")
		assert.Assert(t, cert != nil)
	})

	t.Run("cached certificate is returned when the host is set", func(t *testing.T) {
		certCache := NewCertCache(testCACertPEM, testCAKeyPEM)
		_, err := certCache.AddHost("test-host")
		assert.NilError(t, err)

		cert, err := certCache.Certificate()

		assert.NilError(t, err)
		assert.Equal(t, len(certCache.hosts), 1)
		assert.Equal(t, certCache.hosts[0], "test-host")

		parsedCert, err := x509.ParseCertificate(cert.Certificate[0])
		assert.NilError(t, err)
		assert.Equal(t, len(parsedCert.DNSNames), 1)
		assert.Equal(t, parsedCert.DNSNames[0], "test-host")
	})
}
