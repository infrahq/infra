package engine

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"errors"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/infrahq/infra/internal/registry"
	"github.com/stretchr/testify/assert"
	"gopkg.in/square/go-jose.v2"
	"gopkg.in/square/go-jose.v2/jwt"
)

func TestJWTMiddlewareNoAuthHeader(t *testing.T) {
	getJwkFunc := func() (*jose.JSONWebKey, error) {
		return &jose.JSONWebKey{}, nil
	}

	emptyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	rr := httptest.NewRecorder()

	handler := jwtMiddleware("k8s", getJwkFunc, emptyHandler)

	req, err := http.NewRequestWithContext(context.Background(), "GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Result().StatusCode)
}

func TestJWTMiddlewareNoToken(t *testing.T) {
	getJwkFunc := func() (*jose.JSONWebKey, error) {
		return &jose.JSONWebKey{}, nil
	}

	emptyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	rr := httptest.NewRecorder()

	handler := jwtMiddleware("k8s", getJwkFunc, emptyHandler)

	req, err := http.NewRequestWithContext(context.Background(), "GET", "/", nil)
	req.Header.Set("Authorization", "username:password")
	if err != nil {
		t.Fatal(err)
	}

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Result().StatusCode)
}

func TestJWTMiddlewareInvalidJWKs(t *testing.T) {
	getJwkFunc := func() (*jose.JSONWebKey, error) {
		return nil, errors.New("could not fetch JWKs")
	}

	emptyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	rr := httptest.NewRecorder()

	handler := jwtMiddleware("k8s", getJwkFunc, emptyHandler)

	req, err := http.NewRequestWithContext(context.Background(), "GET", "/", nil)
	req.Header.Set("Authorization", "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwiZW1haWwiOiJ0ZXN0QHRlc3QuY29tIiwiaWF0IjoxNTE2MjM5MDIyfQ.j7o5o8GBkybaYXdFJIi8O6mPF50E-gJWZ3reLfMQD68")
	if err != nil {
		t.Fatal(err)
	}

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Result().StatusCode)
}

func generateJWK() (pub *jose.JSONWebKey, priv *jose.JSONWebKey, err error) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, err
	}

	priv = &jose.JSONWebKey{Key: key, KeyID: "", Algorithm: string(jose.RS256), Use: "sig"}
	thumb, err := priv.Thumbprint(crypto.SHA256)
	if err != nil {
		return nil, nil, err
	}
	kid := base64.URLEncoding.EncodeToString(thumb)
	priv.KeyID = kid
	pub = &jose.JSONWebKey{Key: &key.PublicKey, KeyID: kid, Algorithm: string(jose.RS256), Use: "sig"}

	return pub, priv, err
}

func generateJWT(priv *jose.JSONWebKey, expiry time.Time) (string, error) {
	signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.RS256, Key: priv}, (&jose.SignerOptions{}).WithType("JWT"))
	if err != nil {
		return "", err
	}

	cl := jwt.Claims{
		Issuer:   "infra",
		Expiry:   jwt.NewNumericDate(expiry),
		IssuedAt: jwt.NewNumericDate(time.Now()),
	}
	custom := registry.CustomJWTClaims{
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
	if err != nil {
		t.Fatal(err)
	}

	invalidjwt := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwiZW1haWwiOiJ0ZXN0QHRlc3QuY29tIiwiaWF0IjoxNTE2MjM5MDIyfQ.j7o5o8GBkybaYXdFJIi8O6mPF50E-gJWZ3reLfMQD68"

	getJwkFunc := func() (*jose.JSONWebKey, error) {
		return pub, nil
	}

	emptyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	rr := httptest.NewRecorder()

	handler := jwtMiddleware("k8s", getJwkFunc, emptyHandler)

	req, err := http.NewRequestWithContext(context.Background(), "GET", "/", nil)
	req.Header.Set("Authorization", "Bearer "+invalidjwt)
	if err != nil {
		t.Fatal(err)
	}

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Result().StatusCode)
}

func TestJWTMiddlewareExpiredJWT(t *testing.T) {
	pub, priv, err := generateJWK()
	if err != nil {
		t.Fatal(err)
	}

	expiredJWT, err := generateJWT(priv, time.Now().Add(-1*time.Hour))
	if err != nil {
		t.Fatal(err)
	}

	getJwkFunc := func() (*jose.JSONWebKey, error) {
		return pub, nil
	}

	emptyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	rr := httptest.NewRecorder()

	handler := jwtMiddleware("k8s", getJwkFunc, emptyHandler)

	req, err := http.NewRequestWithContext(context.Background(), "GET", "/", nil)
	req.Header.Set("Authorization", "Bearer "+expiredJWT)
	if err != nil {
		t.Fatal(err)
	}

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Result().StatusCode)

	data, err := ioutil.ReadAll(rr.Result().Body)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "expired\n", string(data))
}

func TestJWTMiddlewareWrongHeader(t *testing.T) {
	pub, priv, err := generateJWK()
	if err != nil {
		t.Fatal(err)
	}

	expiredJWT, err := generateJWT(priv, time.Now().Add(-1*time.Hour))
	if err != nil {
		t.Fatal(err)
	}

	getJwkFunc := func() (*jose.JSONWebKey, error) {
		return pub, nil
	}

	emptyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	rr := httptest.NewRecorder()

	handler := jwtMiddleware("k8s", getJwkFunc, emptyHandler)

	req, err := http.NewRequestWithContext(context.Background(), "GET", "/", nil)
	req.Header.Set("Authorization", "Bearer "+expiredJWT)
	if err != nil {
		t.Fatal(err)
	}

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Result().StatusCode)
}

func TestJWTMiddlewareWrongDestination(t *testing.T) {
	pub, priv, err := generateJWK()
	if err != nil {
		t.Fatal(err)
	}

	validJWT, err := generateJWT(priv, time.Now().Add(3*time.Hour))
	if err != nil {
		t.Fatal(err)
	}

	getJwkFunc := func() (*jose.JSONWebKey, error) {
		return pub, nil
	}

	emptyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	rr := httptest.NewRecorder()

	handler := jwtMiddleware("anotherDestination", getJwkFunc, emptyHandler)

	req, err := http.NewRequestWithContext(context.Background(), "GET", "/", nil)
	req.Header.Set("Authorization", "Bearer "+validJWT)
	if err != nil {
		t.Fatal(err)
	}

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Result().StatusCode)
}

func TestJWTMiddlewareValidJWTSetsEmail(t *testing.T) {
	pub, priv, err := generateJWK()
	if err != nil {
		t.Fatal(err)
	}

	validJWT, err := generateJWT(priv, time.Now().Add(3*time.Hour))
	if err != nil {
		t.Fatal(err)
	}

	getJwkFunc := func() (*jose.JSONWebKey, error) {
		return pub, nil
	}

	emptyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		email, ok := r.Context().Value(HttpContextKeyEmail{}).(string)
		if !ok {
			t.Fatal("could not parse email")
		}
		assert.Equal(t, "test@test.com", email)

		w.WriteHeader(http.StatusOK)
	})

	rr := httptest.NewRecorder()

	handler := jwtMiddleware("k8s", getJwkFunc, emptyHandler)

	req, err := http.NewRequestWithContext(context.Background(), "GET", "/", nil)
	req.Header.Set("Authorization", "Bearer "+validJWT)
	if err != nil {
		t.Fatal(err)
	}

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Result().StatusCode)
}

func TestProxyHandler(t *testing.T) {
	pub, priv, err := generateJWK()
	if err != nil {
		t.Fatal(err)
	}

	validJWT, err := generateJWT(priv, time.Now().Add(3*time.Hour))
	if err != nil {
		t.Fatal(err)
	}

	getJwkFunc := func() (*jose.JSONWebKey, error) {
		return pub, nil
	}

	emptyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		email, ok := r.Context().Value(HttpContextKeyEmail{}).(string)
		if !ok {
			t.Fatal("could not parse email")
		}
		assert.Equal(t, "test@test.com", email)

		w.WriteHeader(http.StatusOK)
	})

	rr := httptest.NewRecorder()

	handler := jwtMiddleware("k8s", getJwkFunc, emptyHandler)

	req, err := http.NewRequestWithContext(context.Background(), "GET", "/", nil)
	req.Header.Set("Authorization", "Bearer "+validJWT)
	if err != nil {
		t.Fatal(err)
	}

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Result().StatusCode)
}
