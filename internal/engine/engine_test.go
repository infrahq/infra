package engine

import (
	"context"
	"crypto"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gopkg.in/square/go-jose.v2"
	"gopkg.in/square/go-jose.v2/jwt"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/access"
)

func TestJWTMiddlewareNoAuthHeader(t *testing.T) {
	emptyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	rr := httptest.NewRecorder()

	handler := jwtMiddleware(emptyHandler, "k8s", "k8s", func() (*jose.JSONWebKey, error) {
		return &jose.JSONWebKey{}, nil
	})

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
	require.NoError(t, err)

	handler.ServeHTTP(rr, req)

	res := rr.Result()
	defer res.Body.Close()

	require.Equal(t, http.StatusUnauthorized, res.StatusCode)
}

func TestJWTMiddlewareNoToken(t *testing.T) {
	emptyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	rr := httptest.NewRecorder()

	handler := jwtMiddleware(emptyHandler, "k8s", "k8s", func() (*jose.JSONWebKey, error) {
		return &jose.JSONWebKey{}, nil
	})

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
	require.NoError(t, err)

	req.Header.Set("Authorization", "username:password")

	handler.ServeHTTP(rr, req)

	res := rr.Result()
	defer res.Body.Close()

	require.Equal(t, http.StatusUnauthorized, res.StatusCode)
}

func TestJWTMiddlewareInvalidJWKs(t *testing.T) {
	emptyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	rr := httptest.NewRecorder()

	handler := jwtMiddleware(emptyHandler, "k8s", "k8s", func() (*jose.JSONWebKey, error) {
		return nil, errors.New("could not fetch JWKs")
	})

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
	require.NoError(t, err)

	req.Header.Set("Authorization", "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwiZW1haWwiOiJ0ZXN0QHRlc3QuY29tIiwiaWF0IjoxNTE2MjM5MDIyfQ.j7o5o8GBkybaYXdFJIi8O6mPF50E-gJWZ3reLfMQD68")

	handler.ServeHTTP(rr, req)

	res := rr.Result()
	defer res.Body.Close()

	require.Equal(t, http.StatusUnauthorized, res.StatusCode)
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

func generateJWT(priv *jose.JSONWebKey, expiry time.Time) (string, error) {
	signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.EdDSA, Key: priv}, (&jose.SignerOptions{}).WithType("JWT"))
	if err != nil {
		return "", err
	}

	cl := jwt.Claims{
		Issuer:   "InfraHQ",
		Expiry:   jwt.NewNumericDate(expiry),
		IssuedAt: jwt.NewNumericDate(time.Now()),
	}
	custom := access.CustomJWTClaims{
		Email:       "test@test.com",
		Nonce:       "randomstring",
		Destination: "k8s",
	}

	raw, err := jwt.Signed(signer).Claims(cl).Claims(custom).CompactSerialize()
	if err != nil {
		return "", err
	}

	return raw, nil
}

func TestJWTMiddlewareInvalidJWT(t *testing.T) {
	pub, _, err := generateJWK()
	require.NoError(t, err)

	invalidjwt := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwiZW1haWwiOiJ0ZXN0QHRlc3QuY29tIiwiaWF0IjoxNTE2MjM5MDIyfQ.j7o5o8GBkybaYXdFJIi8O6mPF50E-gJWZ3reLfMQD68"

	emptyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	rr := httptest.NewRecorder()

	handler := jwtMiddleware(emptyHandler, "k8s", "k8s", func() (*jose.JSONWebKey, error) {
		return pub, nil
	})

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
	require.NoError(t, err)

	req.Header.Set("Authorization", "Bearer "+invalidjwt)

	handler.ServeHTTP(rr, req)

	res := rr.Result()
	defer res.Body.Close()

	require.Equal(t, http.StatusUnauthorized, res.StatusCode)
}

func TestJWTMiddlewareExpiredJWT(t *testing.T) {
	pub, priv, err := generateJWK()
	require.NoError(t, err)

	expiredJWT, err := generateJWT(priv, time.Now().Add(-1*time.Hour))
	require.NoError(t, err)

	emptyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	rr := httptest.NewRecorder()

	handler := jwtMiddleware(emptyHandler, "k8s", "k8s", func() (*jose.JSONWebKey, error) {
		return pub, nil
	})

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
	require.NoError(t, err)

	req.Header.Set("Authorization", "Bearer "+expiredJWT)

	handler.ServeHTTP(rr, req)

	res := rr.Result()
	defer res.Body.Close()

	require.Equal(t, http.StatusUnauthorized, res.StatusCode)

	data, err := ioutil.ReadAll(res.Body)
	require.NoError(t, err)

	require.Equal(t, "expired\n", string(data))
}

func TestJWTMiddlewareWrongHeader(t *testing.T) {
	pub, priv, err := generateJWK()
	require.NoError(t, err)

	expiredJWT, err := generateJWT(priv, time.Now().Add(-1*time.Hour))
	require.NoError(t, err)

	emptyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	rr := httptest.NewRecorder()

	handler := jwtMiddleware(emptyHandler, "k8s", "k8s", func() (*jose.JSONWebKey, error) {
		return pub, nil
	})

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
	require.NoError(t, err)

	req.Header.Set("Authorization", "Bearer "+expiredJWT)

	handler.ServeHTTP(rr, req)

	res := rr.Result()
	defer res.Body.Close()

	require.Equal(t, http.StatusUnauthorized, res.StatusCode)
}

func TestJWTMiddlewareWrongDestination(t *testing.T) {
	pub, priv, err := generateJWK()
	require.NoError(t, err)

	validJWT, err := generateJWT(priv, time.Now().Add(3*time.Hour))
	require.NoError(t, err)

	emptyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	rr := httptest.NewRecorder()

	handler := jwtMiddleware(emptyHandler, "anotherdestination", "anotherDestination", func() (*jose.JSONWebKey, error) {
		return pub, nil
	})

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
	require.NoError(t, err)

	req.Header.Set("Authorization", "Bearer "+validJWT)

	handler.ServeHTTP(rr, req)

	res := rr.Result()
	defer res.Body.Close()

	require.Equal(t, http.StatusUnauthorized, res.StatusCode)
}

func TestJWTMiddlewareValidJWTSetsEmail(t *testing.T) {
	pub, priv, err := generateJWK()
	require.NoError(t, err)

	validJWT, err := generateJWT(priv, time.Now().Add(3*time.Hour))
	require.NoError(t, err)

	emptyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		email, ok := r.Context().Value(internal.HttpContextKeyEmail{}).(string)
		require.True(t, ok)
		require.Equal(t, "test@test.com", email)

		w.WriteHeader(http.StatusOK)
	})

	rr := httptest.NewRecorder()

	handler := jwtMiddleware(emptyHandler, "k8s", "k8s", func() (*jose.JSONWebKey, error) {
		return pub, nil
	})

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
	require.NoError(t, err)

	req.Header.Set("Authorization", "Bearer "+validJWT)

	handler.ServeHTTP(rr, req)

	res := rr.Result()
	defer res.Body.Close()

	require.Equal(t, http.StatusOK, res.StatusCode)
}

func TestProxyHandler(t *testing.T) {
	pub, priv, err := generateJWK()
	require.NoError(t, err)

	validJWT, err := generateJWT(priv, time.Now().Add(3*time.Hour))
	require.NoError(t, err)

	emptyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		email, ok := r.Context().Value(internal.HttpContextKeyEmail{}).(string)
		require.True(t, ok)
		require.Equal(t, "test@test.com", email)

		w.WriteHeader(http.StatusOK)
	})

	rr := httptest.NewRecorder()

	handler := jwtMiddleware(emptyHandler, "k8s", "k8s", func() (*jose.JSONWebKey, error) {
		return pub, nil
	})

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
	require.NoError(t, err)

	req.Header.Set("Authorization", "Bearer "+validJWT)

	handler.ServeHTTP(rr, req)

	res := rr.Result()
	defer res.Body.Close()

	require.Equal(t, http.StatusOK, res.StatusCode)
}
