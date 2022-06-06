package connector

import (
	"crypto"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"gopkg.in/square/go-jose.v2"
	"gopkg.in/square/go-jose.v2/jwt"
	"gotest.tools/v3/assert"

	"github.com/infrahq/infra/internal/claims"
)

func TestJWTMiddlewareNoAuthHeader(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/apis", nil)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = r

	handler := jwtMiddleware(func() (*jose.JSONWebKey, error) {
		return &jose.JSONWebKey{}, nil
	})

	handler(c)

	assert.Equal(t, http.StatusUnauthorized, c.Writer.Status())
}

func TestJWTMiddlewareNoToken(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/apis", nil)
	r.Header.Set("Authorization", "username:password")

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = r

	handler := jwtMiddleware(func() (*jose.JSONWebKey, error) {
		return &jose.JSONWebKey{}, nil
	})

	handler(c)

	assert.Equal(t, http.StatusUnauthorized, c.Writer.Status())
}

func TestJWTMiddlewareInvalidJWKs(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/apis", nil)
	r.Header.Set("Authorization", "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwiZW1haWwiOiJ0ZXN0QHRlc3QuY29tIiwiaWF0IjoxNTE2MjM5MDIyfQ.j7o5o8GBkybaYXdFJIi8O6mPF50E-gJWZ3reLfMQD68")

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = r

	handler := jwtMiddleware(func() (*jose.JSONWebKey, error) {
		return nil, errors.New("could not fetch JWKs")
	})

	handler(c)

	assert.Equal(t, http.StatusUnauthorized, c.Writer.Status())
}

func generateJWK() (pub *jose.JSONWebKey, priv *jose.JSONWebKey, err error) {
	pubkey, key, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, err
	}

	priv = &jose.JSONWebKey{Key: key, KeyID: "", Algorithm: string(jose.ED25519), Use: "sig"}

	thumb, err := priv.Thumbprint(crypto.SHA256)
	if err != nil {
		return nil, nil, err
	}

	kid := base64.URLEncoding.EncodeToString(thumb)
	priv.KeyID = kid
	pub = &jose.JSONWebKey{Key: pubkey, KeyID: kid, Algorithm: string(jose.ED25519), Use: "sig"}

	return pub, priv, err
}

func TestJWTMiddlewareInvalidJWT(t *testing.T) {
	pub, _, err := generateJWK()
	assert.NilError(t, err)

	r := httptest.NewRequest(http.MethodGet, "/apis", nil)
	r.Header.Set("Authorization", "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwiZW1haWwiOiJ0ZXN0QHRlc3QuY29tIiwiaWF0IjoxNTE2MjM5MDIyfQ.j7o5o8GBkybaYXdFJIi8O6mPF50E-gJWZ3reLfMQD68")

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = r

	handler := jwtMiddleware(func() (*jose.JSONWebKey, error) {
		return pub, nil
	})

	handler(c)

	assert.Equal(t, http.StatusUnauthorized, c.Writer.Status())
}

func generateJWT(priv *jose.JSONWebKey, email string, expiry time.Time) (string, error) {
	signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.EdDSA, Key: priv}, (&jose.SignerOptions{}).WithType("JWT"))
	if err != nil {
		return "", err
	}

	cl := jwt.Claims{
		Issuer:   "InfraHQ",
		Expiry:   jwt.NewNumericDate(expiry),
		IssuedAt: jwt.NewNumericDate(time.Now()),
	}

	custom := claims.Custom{
		Name:   email,
		Groups: []string{"developers"},
		Nonce:  "randomstring",
	}

	raw, err := jwt.Signed(signer).Claims(cl).Claims(custom).CompactSerialize()
	if err != nil {
		return "", err
	}

	return raw, nil
}

func TestJWTMiddlewareExpiredJWT(t *testing.T) {
	pub, sec, err := generateJWK()
	assert.NilError(t, err)

	jwt, err := generateJWT(sec, "test@example.com", time.Now().Add(-1*time.Hour))
	assert.NilError(t, err)

	r := httptest.NewRequest(http.MethodGet, "/apis", nil)
	r.Header.Set("Authorization", fmt.Sprintf("Bearer %s", jwt))

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = r

	handler := jwtMiddleware(func() (*jose.JSONWebKey, error) {
		return pub, nil
	})

	handler(c)

	assert.Equal(t, http.StatusUnauthorized, c.Writer.Status())
}

func TestJWTMiddlewareValidJWT(t *testing.T) {
	pub, sec, err := generateJWK()
	assert.NilError(t, err)

	jwt, err := generateJWT(sec, "test@example.com", time.Now().Add(1*time.Hour))
	assert.NilError(t, err)

	r := httptest.NewRequest(http.MethodGet, "/apis", nil)
	r.Header.Set("Authorization", fmt.Sprintf("Bearer %s", jwt))

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = r

	handler := jwtMiddleware(func() (*jose.JSONWebKey, error) {
		return pub, nil
	})

	handler(c)

	assert.Equal(t, http.StatusOK, c.Writer.Status())

	name, nameExists := c.Get("name")
	assert.Assert(t, nameExists)
	assert.Equal(t, "test@example.com", name)

	groups, groupsExists := c.Get("groups")
	assert.Assert(t, groupsExists)
	assert.DeepEqual(t, []string{"developers"}, groups)
}

func TestCertificate(t *testing.T) {
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
		certCache.AddHost("test-host")

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
