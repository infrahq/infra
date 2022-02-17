package engine

import (
	"fmt"
	"crypto"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"gopkg.in/square/go-jose.v2"
	"gopkg.in/square/go-jose.v2/jwt"

	"github.com/infrahq/infra/internal/claims"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestJWTMiddlewareNoAuthHeader(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/apis", nil)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = r

	handler := jwtMiddleware(func() (*jose.JSONWebKey, error) {
		return &jose.JSONWebKey{}, nil
	})

	handler(c)

	require.Equal(t, http.StatusUnauthorized, c.Writer.Status())
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

	require.Equal(t, http.StatusUnauthorized, c.Writer.Status())
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

	require.Equal(t, http.StatusUnauthorized, c.Writer.Status())
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
	require.NoError(t, err)

	r := httptest.NewRequest(http.MethodGet, "/apis", nil)
	r.Header.Set("Authorization", "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwiZW1haWwiOiJ0ZXN0QHRlc3QuY29tIiwiaWF0IjoxNTE2MjM5MDIyfQ.j7o5o8GBkybaYXdFJIi8O6mPF50E-gJWZ3reLfMQD68")

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = r

	handler := jwtMiddleware(func() (*jose.JSONWebKey, error) {
		return pub, nil
	})

	handler(c)

	require.Equal(t, http.StatusUnauthorized, c.Writer.Status())
}

func generateJWT(priv *jose.JSONWebKey, email, machineName string, expiry time.Time) (string, error) {
	signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.EdDSA, Key: priv}, (&jose.SignerOptions{}).WithType("JWT"))
	if err != nil {
		return "", err
	}

	cl := jwt.Claims{
		Issuer:   "InfraHQ",
		Expiry:   jwt.NewNumericDate(expiry),
		IssuedAt: jwt.NewNumericDate(time.Now()),
	}

	var custom claims.Custom
	if email != "" {
		custom = claims.Custom{
			Email:  email,
			Groups: []string{"developers"},
			Nonce:  "randomstring",
		}
	} else {
		custom = claims.Custom{
			Machine: machineName,
			Nonce:   "randomstring",
		}
	}

	raw, err := jwt.Signed(signer).Claims(cl).Claims(custom).CompactSerialize()
	if err != nil {
		return "", err
	}

	return raw, nil
}

func TestJWTMiddlewareExpiredJWT(t *testing.T) {
	pub, sec, err := generateJWK()
	require.NoError(t, err)

	jwt, err := generateJWT(sec, "test@example.com", "", time.Now().Add(-1*time.Hour))
	require.NoError(t, err)

	r := httptest.NewRequest(http.MethodGet, "/apis", nil)
	r.Header.Set("Authorization", fmt.Sprintf("Bearer %s", jwt))

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = r

	handler := jwtMiddleware(func() (*jose.JSONWebKey, error) {
		return pub, nil
	})

	handler(c)

	require.Equal(t, http.StatusUnauthorized, c.Writer.Status())
}

func TestJWTMiddlewareValidJWT(t *testing.T) {
	pub, sec, err := generateJWK()
	require.NoError(t, err)

	jwt, err := generateJWT(sec, "test@example.com", "", time.Now().Add(1*time.Hour))
	require.NoError(t, err)

	r := httptest.NewRequest(http.MethodGet, "/apis", nil)
	r.Header.Set("Authorization", fmt.Sprintf("Bearer %s", jwt))

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = r

	handler := jwtMiddleware(func() (*jose.JSONWebKey, error) {
		return pub, nil
	})

	handler(c)

	require.Equal(t, http.StatusOK, c.Writer.Status())

	email, emailExists := c.Get("email")
	require.True(t, emailExists)
	require.Equal(t, "test@example.com", email)

	machine, machineExists := c.Get("machine")
	require.True(t, machineExists)
	require.Empty(t, machine)

	groups, groupsExists := c.Get("groups")
	require.True(t, groupsExists)
	require.Equal(t, []string{"developers"}, groups)
}

func TestJWTMiddlewareValidMachineJWT(t *testing.T) {
	pub, sec, err := generateJWK()
	require.NoError(t, err)

	jwt, err := generateJWT(sec, "", "arnold", time.Now().Add(1*time.Hour))
	require.NoError(t, err)

	r := httptest.NewRequest(http.MethodGet, "/apis", nil)
	r.Header.Set("Authorization", fmt.Sprintf("Bearer %s", jwt))

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = r

	handler := jwtMiddleware(func() (*jose.JSONWebKey, error) {
		return pub, nil
	})

	handler(c)

	require.Equal(t, http.StatusOK, c.Writer.Status())

	email, emailExists := c.Get("email")
	require.True(t, emailExists)
	require.Empty(t, email)

	machine, machineExists := c.Get("machine")
	require.True(t, machineExists)
	require.Equal(t, "arnold", machine)

	groups, groupsExists := c.Get("groups")
	require.True(t, groupsExists)
	require.Empty(t, groups)
}
