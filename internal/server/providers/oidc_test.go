package providers

import (
	"context"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"golang.org/x/oauth2"
	"gopkg.in/square/go-jose.v2"
	"gopkg.in/square/go-jose.v2/jwt"
	"gorm.io/gorm"
	"gotest.tools/v3/assert"

	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/internal/testing/patch"
)

type tokenResponse struct {
	code int
	body string
}

// testOIDCServer mocks the expected responses from an OIDC provider
type testOIDCServer struct {
	userInfoResponse string
	tokenResponse    tokenResponse
	signingKey       *rsa.PrivateKey
}

const (
	oktaInvalidClientIDResp = `{
		"errorCode": "invalid_client",
		"errorSummary": "Invalid value for 'client_id' parameter.",
		"errorLink": "invalid_client",
		"errorId": "aaabbb",
		"errorCauses": []
	}`

	//nolint:gosec // this is not an actual secret credential
	oktaInvalidClientSecretResp = `{
		"error": "invalid_client",
		"error_description": "The client secret supplied for a confidential client is invalid."
	}`

	oktaInvalidAuthCodeResp = `{
		"error": "invalid_grant",
		"error_description": "The authorization code is invalid or has expired."
	}`
)

func cmpAPITimeWithThreshold(x, y time.Time) bool {
	if x.IsZero() || y.IsZero() {
		return false
	}
	delta := x.Sub(y)

	threshold := 20 * time.Minute

	return delta <= threshold && delta >= -threshold
}

func setupDB(t *testing.T) *gorm.DB {
	driver, err := data.NewSQLiteDriver("file::memory:")
	assert.NilError(t, err)

	patch.ModelsSymmetricKey(t)
	db, err := data.NewDB(driver, nil)
	assert.NilError(t, err)

	return db
}

func (ts *testOIDCServer) run(t *testing.T, addHandlers func(*testing.T, *http.ServeMux)) string {
	newMux := http.NewServeMux()
	server := httptest.NewTLSServer(newMux)

	// this is the jwks public key that corresponds with _testdata/test-server-sec.key we just loaded
	jwks := `{
		"keys": [
			{
				"kty": "RSA",
				"n": "17B6CrEmJYA2-bPUxaL0OqBg3PWs19ab2DIwrgNc2vPdseV2rUWtXTjVFhVM4cfhuJRULpGJHw8Us3YyJNrIWOaPhOlLCOkrtWK3jCl2xEhgLkpyGxLzsjkSGmOmTNtz44dnxW3WNW-3fKr-aAlGI9O81x3tbU7NnNvvFkotC2HvElQIjISw118C13b9SY1Xc7iKJYcwu6NiSnnkazGGkDLZcR0Ja4lp7Iym8sIbQ72o_mGUUDoCO3CDJePliq4RRKvxCPH2SOuzRpUGwzyK4bfZPXRR17GcOd7DJU6kfGxfX6oAuJCqCEUZCYfQYo6Uj4cBl0BPANlPQuJk7irRvw",
				"e": "AQAB",
				"alg": "RS256",
				"kid": "openid-test",
				"use": "sig"
			}
		]
	}`

	wellKnown := fmt.Sprintf(`{
		"issuer": "%[1]s",
		"authorization_endpoint": "%[1]s/auth",
		"token_endpoint": "%[1]s/token",
		"jwks_uri": "%[1]s/keys",
		"userinfo_endpoint": "%[1]s/userinfo",
		"id_token_signing_alg_values_supported": ["RS256"]
	}`, server.URL)

	// general OIDC endpoints
	newMux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, req *http.Request) {
		_, err := io.WriteString(w, wellKnown)
		if err != nil {
			w.WriteHeader(500)
		}
	})
	newMux.HandleFunc("/keys", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		_, err := io.WriteString(w, jwks)
		if err != nil {
			w.WriteHeader(500)
		}
	})
	newMux.HandleFunc("/token", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(ts.tokenResponse.code)
		_, err := io.WriteString(w, ts.tokenResponse.body)
		if err != nil {
			assert.Check(t, err, "failed to write token response")
		}
	})
	newMux.HandleFunc("/userinfo", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		_, err := io.WriteString(w, ts.userInfoResponse)
		if err != nil {
			w.WriteHeader(500)
		}
	})

	if addHandlers != nil {
		addHandlers(t, newMux)
	}

	t.Cleanup(server.Close)
	return strings.ReplaceAll(server.URL, "https://", "")
}

func loadTestSecretKey(t *testing.T) *rsa.PrivateKey {
	sec, err := ioutil.ReadFile("./_testdata/test-server-sec.key")
	assert.NilError(t, err)

	secBlock, _ := pem.Decode([]byte(sec))
	assert.Assert(t, secBlock != nil)

	rsaSecKey, err := x509.ParsePKCS1PrivateKey(secBlock.Bytes)
	assert.NilError(t, err)

	return rsaSecKey
}

func testTokenResponse(claims jwt.Claims, signingKey *rsa.PrivateKey, email string) (string, error) {
	options := &jose.SignerOptions{}

	signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.SignatureAlgorithm("RS256"), Key: signingKey}, options.WithType("JWT"))
	if err != nil {
		return "", err
	}

	raw := ""
	if email != "" {
		type Custom struct {
			Email string `json:"email"`
		}

		raw, err = jwt.Signed(signer).Claims(claims).Claims(Custom{Email: email}).CompactSerialize()
		if err != nil {
			return "", err
		}
	} else {
		raw, err = jwt.Signed(signer).Claims(claims).CompactSerialize()
		if err != nil {
			return "", err
		}
	}

	return fmt.Sprintf(`{
		"access_token": "eyJhbGciOiJSUzI1NiJ9.eyJ2ZXIiOjEsImlzcyI6Imh0dHA6Ly9yYWluLm9rdGExLmNvbToxODAyIiwiaWF0IjoxNDQ5NjI0MDI2LCJleHAiOjE0NDk2Mjc2MjYsImp0aSI6IlVmU0lURzZCVVNfdHA3N21BTjJxIiwic2NvcGVzIjpbIm9wZW5pZCIsImVtYWlsIl0sImNsaWVudF9pZCI6InVBYXVub2ZXa2FESnh1a0NGZUJ4IiwidXNlcl9pZCI6IjAwdWlkNEJ4WHc2STZUVjRtMGczIn0.HaBu5oQxdVCIvea88HPgr2O5evqZlCT4UXH4UKhJnZ5px-ArNRqwhxXWhHJisslswjPpMkx1IgrudQIjzGYbtLFjrrg2ueiU5-YfmKuJuD6O2yPWGTsV7X6i7ABT6P-t8PRz_RNbk-U1GXWIEkNnEWbPqYDAm_Ofh7iW0Y8WDA5ez1jbtMvd-oXMvJLctRiACrTMLJQ2e5HkbUFxgXQ_rFPNHJbNSUBDLqdi2rg_ND64DLRlXRY7hupNsvWGo0gF4WEUk8IZeaLjKw8UoIs-ETEwJlAMcvkhoVVOsN5dPAaEKvbyvPC1hUGXb4uuThlwdD3ECJrtwgKqLqcWonNtiw",
		"token_type": "Bearer",
		"expires_in": 3600,
		"scope": "openid email",
		"refresh_token": "a9VpZDRCeFh3Nkk2VdY",
		"id_token": "%[1]s"
	}`, raw), nil
}

func setupOIDCTest(t *testing.T, userInfoResp string) (testOIDCServer, context.Context) {
	signingKey := loadTestSecretKey(t)

	server := testOIDCServer{
		signingKey:       signingKey,
		userInfoResponse: userInfoResp,
	}

	// setup a an HTTP client that skips TLS verify for test purposes
	//nolint:forcedtypeassert
	testTransport, ok := http.DefaultTransport.(*http.Transport)
	assert.Assert(t, ok)
	testTransport = testTransport.Clone()
	//nolint:gosec // skipping TLS verify for testing
	testTransport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	client := &http.Client{Transport: testTransport}

	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, client)

	return server, ctx
}

func TestValidate(t *testing.T) {
	server, ctx := setupOIDCTest(t, "")
	serverURL := server.run(t, nil)

	tests := []struct {
		name          string
		provider      OIDC
		tokenResponse tokenResponse
		verifyFunc    func(*testing.T, error)
	}{
		{
			name:     "invalid URL",
			provider: NewOIDC(models.Provider{Kind: models.OIDCKind, URL: "example.com"}, "some_client_secret", "http://localhost:8301"),
			verifyFunc: func(t *testing.T, err error) {
				assert.ErrorIs(t, err, ErrInvalidProviderURL)
			},
		},
		{
			name:     "invalid client ID",
			provider: NewOIDC(models.Provider{Kind: models.OIDCKind, URL: serverURL, ClientID: "invalid-client-id"}, "some_client_secret", "http://localhost:8301"),
			tokenResponse: tokenResponse{
				code: 500,
				body: oktaInvalidClientIDResp,
			},
			verifyFunc: func(t *testing.T, err error) {
				assert.ErrorIs(t, err, ErrInvalidProviderClientID)
			},
		},
		{
			name:     "invalid client secret",
			provider: NewOIDC(models.Provider{Kind: models.OIDCKind, URL: serverURL, ClientID: "client-id"}, "some_client_secret", "http://localhost:8301"),
			tokenResponse: tokenResponse{
				code: 500,
				body: oktaInvalidClientSecretResp,
			},
			verifyFunc: func(t *testing.T, err error) {
				assert.ErrorIs(t, err, ErrInvalidProviderClientSecret)
			},
		},

		{
			name:     "valid provider client",
			provider: NewOIDC(models.Provider{Kind: models.OIDCKind, URL: serverURL, ClientID: "client-id"}, "some_client_secret", "http://localhost:8301"),
			tokenResponse: tokenResponse{
				code: 500,
				body: oktaInvalidAuthCodeResp,
			},
			verifyFunc: func(t *testing.T, err error) {
				assert.NilError(t, err)
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server.tokenResponse = test.tokenResponse
			err := test.provider.Validate(ctx)
			test.verifyFunc(t, err)
		})
	}
}

func TestExchangeAuthCodeForProviderToken(t *testing.T) {
	server, ctx := setupOIDCTest(t, "")
	serverURL := server.run(t, nil)

	tests := []struct {
		name          string
		provider      OIDC
		tokenResponse func(t *testing.T) tokenResponse
		verifyFunc    func(t *testing.T, accessToken, refreshToken string, accessTokenExpiry time.Time, email string, err error)
	}{
		{
			name:     "invalid provider client fails",
			provider: NewOIDC(models.Provider{Kind: models.OIDCKind, URL: serverURL, ClientID: "invalid"}, "invalid", "http://localhost:8301"),
			tokenResponse: func(t *testing.T) tokenResponse {
				return tokenResponse{
					code: 500,
					body: oktaInvalidClientIDResp,
				}
			},
			verifyFunc: func(t *testing.T, accessToken, refreshToken string, accessTokenExpiry time.Time, email string, err error) {
				assert.ErrorContains(t, err, "Invalid value for 'client_id' parameter")
				assert.Equal(t, accessToken, "")
				assert.Equal(t, refreshToken, "")
				assert.Assert(t, accessTokenExpiry.IsZero())
				assert.Equal(t, email, "")
			},
		},
		{
			name:     "invalid auth code fails",
			provider: NewOIDC(models.Provider{Kind: models.OIDCKind, URL: serverURL, ClientID: "client-id"}, "some_client_secret", "http://localhost:8301"),
			tokenResponse: func(t *testing.T) tokenResponse {
				return tokenResponse{
					code: 500,
					body: oktaInvalidAuthCodeResp,
				}
			},
			verifyFunc: func(t *testing.T, accessToken, refreshToken string, accessTokenExpiry time.Time, email string, err error) {
				assert.ErrorContains(t, err, "authorization code is invalid")
				assert.Equal(t, accessToken, "")
				assert.Equal(t, refreshToken, "")
				assert.Assert(t, accessTokenExpiry.IsZero())
				assert.Equal(t, email, "")
			},
		},
		{
			name:     "empty access token response fails",
			provider: NewOIDC(models.Provider{Kind: models.OIDCKind, URL: serverURL, ClientID: "client-id"}, "some_client_secret", "http://localhost:8301"),
			tokenResponse: func(t *testing.T) tokenResponse {
				return tokenResponse{
					code: 200,
					body: `{"access_token": ""}`,
				}
			},
			verifyFunc: func(t *testing.T, accessToken, refreshToken string, accessTokenExpiry time.Time, email string, err error) {
				assert.ErrorContains(t, err, "server response missing access_token")
				assert.Equal(t, accessToken, "")
				assert.Equal(t, refreshToken, "")
				assert.Assert(t, accessTokenExpiry.IsZero())
				assert.Equal(t, email, "")
			},
		},
		{
			name:     "id token issued by a different provider fails",
			provider: NewOIDC(models.Provider{Kind: models.OIDCKind, URL: serverURL, ClientID: "client-id"}, "some_client_secret", "http://localhost:8301"),
			tokenResponse: func(t *testing.T) tokenResponse {
				claims := jwt.Claims{
					Issuer: "unknown-issuer",
				}

				var err error
				body, err := testTokenResponse(claims, server.signingKey, "")
				assert.NilError(t, err)

				return tokenResponse{
					code: 200,
					body: body,
				}
			},
			verifyFunc: func(t *testing.T, accessToken, refreshToken string, accessTokenExpiry time.Time, email string, err error) {
				assert.ErrorContains(t, err, "id token issued by a different provider")
				assert.Equal(t, accessToken, "")
				assert.Equal(t, refreshToken, "")
				assert.Assert(t, accessTokenExpiry.IsZero())
				assert.Equal(t, email, "")
			},
		},
		{
			name:     "id token issued for wrong audience fails",
			provider: NewOIDC(models.Provider{Kind: models.OIDCKind, URL: serverURL, ClientID: "client-id"}, "some_client_secret", "http://localhost:8301"),
			tokenResponse: func(t *testing.T) tokenResponse {
				claims := jwt.Claims{
					Issuer:   "https://" + serverURL,
					Audience: jwt.Audience([]string{"unknown-client"}),
				}

				var err error
				body, err := testTokenResponse(claims, server.signingKey, "")
				assert.NilError(t, err)

				return tokenResponse{
					code: 200,
					body: body,
				}
			},
			verifyFunc: func(t *testing.T, accessToken, refreshToken string, accessTokenExpiry time.Time, email string, err error) {
				assert.ErrorContains(t, err, "expected audience \"client-id\"")
				assert.Equal(t, accessToken, "")
				assert.Equal(t, refreshToken, "")
				assert.Assert(t, accessTokenExpiry.IsZero())
				assert.Equal(t, email, "")
			},
		},
		{
			name:     "expired id token fails",
			provider: NewOIDC(models.Provider{Kind: models.OIDCKind, URL: serverURL, ClientID: "client-id"}, "some_client_secret", "http://localhost:8301"),
			tokenResponse: func(t *testing.T) tokenResponse {
				now := time.Now().UTC()

				claims := jwt.Claims{
					Audience:  jwt.Audience([]string{"client-id"}),
					NotBefore: jwt.NewNumericDate(now.Add(-5 * time.Minute)),
					Expiry:    jwt.NewNumericDate(now.Add(-5 * time.Minute)),
					IssuedAt:  jwt.NewNumericDate(now),
					Issuer:    "https://" + serverURL,
				}

				var err error
				body, err := testTokenResponse(claims, server.signingKey, "")
				assert.NilError(t, err)

				return tokenResponse{
					code: 200,
					body: body,
				}
			},
			verifyFunc: func(t *testing.T, accessToken, refreshToken string, accessTokenExpiry time.Time, email string, err error) {
				assert.ErrorContains(t, err, "token is expired")
				assert.Equal(t, accessToken, "")
				assert.Equal(t, refreshToken, "")
				assert.Assert(t, accessTokenExpiry.IsZero())
				assert.Equal(t, email, "")
			},
		},
		{
			name:     "id token without email claim fails",
			provider: NewOIDC(models.Provider{Kind: models.OIDCKind, URL: serverURL, ClientID: "client-id"}, "some_client_secret", "http://localhost:8301"),
			tokenResponse: func(t *testing.T) tokenResponse {
				now := time.Now().UTC()

				claims := jwt.Claims{
					Audience:  jwt.Audience([]string{"client-id"}),
					NotBefore: jwt.NewNumericDate(now.Add(-5 * time.Minute)), // adjust for clock drift
					Expiry:    jwt.NewNumericDate(now.Add(5 * time.Minute)),
					IssuedAt:  jwt.NewNumericDate(now),
					Issuer:    "https://" + serverURL,
				}

				var err error
				body, err := testTokenResponse(claims, server.signingKey, "")
				assert.NilError(t, err)

				return tokenResponse{
					code: 200,
					body: body,
				}
			},
			verifyFunc: func(t *testing.T, accessToken, refreshToken string, accessTokenExpiry time.Time, email string, err error) {
				assert.ErrorContains(t, err, "validation for 'Email' failed")
				assert.Equal(t, accessToken, "")
				assert.Equal(t, refreshToken, "")
				assert.Assert(t, accessTokenExpiry.IsZero())
				assert.Equal(t, email, "")
			},
		},
		{
			name:     "empty email claim fails",
			provider: NewOIDC(models.Provider{Kind: models.OIDCKind, URL: serverURL, ClientID: "client-id"}, "some_client_secret", "http://localhost:8301"),
			tokenResponse: func(t *testing.T) tokenResponse {
				now := time.Now().UTC()

				claims := jwt.Claims{
					Audience:  jwt.Audience([]string{"client-id"}),
					NotBefore: jwt.NewNumericDate(now.Add(-5 * time.Minute)), // adjust for clock drift
					Expiry:    jwt.NewNumericDate(now.Add(5 * time.Minute)),
					IssuedAt:  jwt.NewNumericDate(now),
					Issuer:    "https://" + serverURL,
				}

				var err error
				body, err := testTokenResponse(claims, server.signingKey, " ")
				assert.NilError(t, err)

				return tokenResponse{
					code: 200,
					body: body,
				}
			},
			verifyFunc: func(t *testing.T, accessToken, refreshToken string, accessTokenExpiry time.Time, email string, err error) {
				assert.ErrorContains(t, err, "'Email' failed on the 'excludesall' tag")
				assert.Equal(t, accessToken, "")
				assert.Equal(t, refreshToken, "")
				assert.Assert(t, accessTokenExpiry.IsZero())
				assert.Equal(t, email, "")
			},
		},
		{
			name:     "valid id token is successful",
			provider: NewOIDC(models.Provider{Kind: models.OIDCKind, URL: serverURL, ClientID: "client-id"}, "some_client_secret", "http://localhost:8301"),
			tokenResponse: func(t *testing.T) tokenResponse {
				now := time.Now().UTC()

				claims := jwt.Claims{
					Audience:  jwt.Audience([]string{"client-id"}),
					NotBefore: jwt.NewNumericDate(now.Add(-5 * time.Minute)), // adjust for clock drift
					Expiry:    jwt.NewNumericDate(now.Add(5 * time.Minute)),
					IssuedAt:  jwt.NewNumericDate(now),
					Issuer:    "https://" + serverURL,
				}

				var err error
				body, err := testTokenResponse(claims, server.signingKey, "hello@example.com")
				assert.NilError(t, err)

				return tokenResponse{
					code: 200,
					body: body,
				}
			},
			verifyFunc: func(t *testing.T, accessToken, refreshToken string, accessTokenExpiry time.Time, email string, err error) {
				assert.NilError(t, err)
				assert.Assert(t, accessToken != "")
				assert.Assert(t, refreshToken != "")
				assert.Assert(t, !accessTokenExpiry.IsZero())
				assert.Equal(t, email, "hello@example.com")
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server.tokenResponse = test.tokenResponse(t)
			accToken, refToken, accTokenExp, email, err := test.provider.ExchangeAuthCodeForProviderTokens(ctx, "some-auth-code")
			test.verifyFunc(t, accToken, refToken, accTokenExp, email, err)
		})
	}
}

func TestRefreshAccessToken(t *testing.T) {
	server, ctx := setupOIDCTest(t, "")
	serverURL := server.run(t, nil)
	provider := NewOIDC(models.Provider{Kind: models.OIDCKind, URL: serverURL, ClientID: "whatever"}, "secret", "http://localhost:8301")

	now := time.Now().UTC()

	claims := jwt.Claims{
		Audience:  jwt.Audience([]string{"client-id"}),
		NotBefore: jwt.NewNumericDate(now.Add(-5 * time.Minute)), // adjust for clock drift
		Expiry:    jwt.NewNumericDate(now.Add(5 * time.Minute)),
		IssuedAt:  jwt.NewNumericDate(now),
		Issuer:    serverURL,
	}

	body, err := testTokenResponse(claims, server.signingKey, "hello@example.com")
	assert.NilError(t, err)

	tests := []struct {
		name          string
		providerUser  *models.ProviderUser
		tokenResponse tokenResponse
		verifyFunc    func(t *testing.T, accessToken string, expiry *time.Time, err error)
	}{
		{
			name: "invalid/expired refresh token fails",
			providerUser: &models.ProviderUser{
				AccessToken:  models.EncryptedAtRest("aaa"),
				RefreshToken: models.EncryptedAtRest("bbb"),
				ExpiresAt:    time.Now().UTC().Add(-5 * time.Minute),
			},
			tokenResponse: tokenResponse{
				code: 403,
				body: "",
			},
			verifyFunc: func(t *testing.T, accessToken string, expiry *time.Time, err error) {
				assert.ErrorContains(t, err, "cannot fetch token")
			},
		},
		{
			name: "valid access token is not refreshed",
			providerUser: &models.ProviderUser{
				AccessToken:  models.EncryptedAtRest("aaa"),
				RefreshToken: models.EncryptedAtRest("bbb"),
				ExpiresAt:    time.Now().UTC().Add(5 * time.Minute),
			},
			verifyFunc: func(t *testing.T, accessToken string, expiry *time.Time, err error) {
				assert.NilError(t, err)
				assert.Equal(t, accessToken, "aaa")
			},
		},
		{
			name: "expired access token is refreshed",
			providerUser: &models.ProviderUser{
				AccessToken:  models.EncryptedAtRest("aaa"),
				RefreshToken: models.EncryptedAtRest("bbb"),
				ExpiresAt:    time.Now().UTC().Add(-5 * time.Minute),
			},
			tokenResponse: tokenResponse{
				code: 200,
				body: body,
			},
			verifyFunc: func(t *testing.T, accessToken string, expiry *time.Time, err error) {
				assert.NilError(t, err)
				assert.Assert(t, accessToken != "aaa")
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server.tokenResponse = test.tokenResponse
			accessToken, exp, err := provider.RefreshAccessToken(ctx, test.providerUser)
			test.verifyFunc(t, accessToken, exp, err)
		})
	}
}

func TestSyncProviderUser(t *testing.T) {
	db := setupDB(t)

	provider := &models.Provider{
		Name: "mockta",
		Kind: models.OktaKind,
	}

	err := data.CreateProvider(db, provider)
	assert.NilError(t, err)

	tests := []struct {
		name              string
		setupProviderUser func(t *testing.T) *models.Identity
		infoResponse      string
		verifyFunc        func(t *testing.T, err error, user *models.Identity)
	}{
		{
			name: "invalid/expired access token is updated",
			setupProviderUser: func(t *testing.T) *models.Identity {
				user := &models.Identity{
					Name: "hello@example.com",
				}

				err = data.CreateIdentity(db, user)
				assert.NilError(t, err)

				pu := &models.ProviderUser{
					ProviderID: provider.ID,
					IdentityID: user.ID,

					Email:        user.Name,
					RedirectURL:  "http://example.com",
					AccessToken:  models.EncryptedAtRest("aaa"),
					RefreshToken: models.EncryptedAtRest("bbb"),
					ExpiresAt:    time.Now().UTC().Add(-5 * time.Minute),
					LastUpdate:   time.Now().UTC().Add(-1 * time.Hour),
				}

				err = data.UpdateProviderUser(db, pu)
				assert.NilError(t, err)

				return user
			},
			infoResponse: `{
				"email": "hello@example.com",
				"groups": [
					"Everyone",
					"Developers"
				]
			}`,
			verifyFunc: func(t *testing.T, err error, user *models.Identity) {
				assert.NilError(t, err)

				pu, err := data.GetProviderUser(db, provider.ID, user.ID)
				assert.NilError(t, err)

				assert.Assert(t, string(pu.AccessToken) != "aaa")
				assert.Equal(t, string(pu.RefreshToken), "bbb")
				assert.Assert(t, cmpAPITimeWithThreshold(pu.ExpiresAt, time.Now().UTC().Add(1*time.Hour)))
				assert.Assert(t, cmpAPITimeWithThreshold(pu.LastUpdate, time.Now().UTC()))
			},
		},
		{
			name: "groups are updated to match user info",
			setupProviderUser: func(t *testing.T) *models.Identity {
				user := &models.Identity{
					Name: "sync@example.com",
				}

				err = data.CreateIdentity(db, user)
				assert.NilError(t, err)

				pu := &models.ProviderUser{
					ProviderID: provider.ID,
					IdentityID: user.ID,

					Email:        user.Name,
					RedirectURL:  "http://example.com",
					AccessToken:  models.EncryptedAtRest("aaa"),
					RefreshToken: models.EncryptedAtRest("bbb"),
					ExpiresAt:    time.Now().UTC().Add(5 * time.Minute),
					LastUpdate:   time.Now().UTC().Add(-1 * time.Hour),
				}

				err = data.UpdateProviderUser(db, pu)
				assert.NilError(t, err)

				return user
			},
			infoResponse: `{
				"email": "sync@example.com",
				"groups": [
					"Everyone",
					"Developers"
				]
			}`,
			verifyFunc: func(t *testing.T, err error, user *models.Identity) {
				assert.NilError(t, err)

				pu, err := data.GetProviderUser(db, provider.ID, user.ID)
				assert.NilError(t, err)
				assert.Assert(t, cmpAPITimeWithThreshold(pu.LastUpdate, time.Now().UTC()))

				assert.Assert(t, len(pu.Groups) == 2)

				puGroups := make(map[string]bool)
				for _, g := range pu.Groups {
					puGroups[g] = true
				}

				assert.Assert(t, puGroups["Everyone"])
				assert.Assert(t, puGroups["Developers"])

				// check that the direct user-to-group relation was updated
				storedGroups, err := data.ListGroups(db, data.ByGroupMember(pu.IdentityID))
				assert.NilError(t, err)

				userGroups := make(map[string]bool)
				for _, g := range storedGroups {
					userGroups[g.Name] = true
				}

				assert.Assert(t, userGroups["Everyone"])
				assert.Assert(t, userGroups["Developers"])
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server, ctx := setupOIDCTest(t, test.infoResponse)
			serverURL := server.run(t, nil)
			oidc := NewOIDC(models.Provider{Kind: models.OIDCKind, URL: serverURL, ClientID: "invalid"}, "invalid", "http://localhost:8301")

			now := time.Now().UTC()

			claims := jwt.Claims{
				Audience:  jwt.Audience([]string{"client-id"}),
				NotBefore: jwt.NewNumericDate(now.Add(-5 * time.Minute)), // adjust for clock drift
				Expiry:    jwt.NewNumericDate(now.Add(5 * time.Minute)),
				IssuedAt:  jwt.NewNumericDate(now),
				Issuer:    serverURL,
			}

			body, err := testTokenResponse(claims, server.signingKey, "hello@example.com")
			assert.NilError(t, err)

			server.tokenResponse = tokenResponse{
				code: 200,
				body: body,
			}

			user := test.setupProviderUser(t)
			server.userInfoResponse = test.infoResponse
			err = oidc.SyncProviderUser(ctx, db, user, provider)
			test.verifyFunc(t, err, user)
		})
	}
}

func TestGetUserInfo(t *testing.T) {
	tests := []struct {
		name         string
		infoResponse string
		verifyFunc   func(t *testing.T, info *InfoClaims, err error)
	}{
		{
			name:         "empty user info response fails",
			infoResponse: "",
			verifyFunc: func(t *testing.T, info *InfoClaims, err error) {
				assert.ErrorContains(t, err, "failed to decode userinfo")
				assert.Assert(t, info == nil)
			},
		},
		{
			name: "user info with no name or email fails",
			infoResponse: `{
					"groups": []
				}`,
			verifyFunc: func(t *testing.T, info *InfoClaims, err error) {
				assert.ErrorContains(t, err, "required_without")
				assert.Assert(t, info == nil)
			},
		},
		{
			name: "user info with no name succeeds",
			infoResponse: `{
					"email": "hello@example.com"
				}`,
			verifyFunc: func(t *testing.T, info *InfoClaims, err error) {
				assert.NilError(t, err, "required_without")
				assert.Equal(t, info.Email, "hello@example.com")
				assert.Equal(t, info.Name, "")
				assert.Assert(t, info.Groups == nil)
			},
		},
		{
			name: "user info with no email succeeds",
			infoResponse: `{
					"name": "hello"
				}`,
			verifyFunc: func(t *testing.T, info *InfoClaims, err error) {
				assert.NilError(t, err, "required_without")
				assert.Equal(t, info.Email, "")
				assert.Equal(t, info.Name, "hello")
				assert.Assert(t, info.Groups == nil)
			},
		},
		{
			name: "full user info response succeeds",
			infoResponse: `{
					"name": "hello",
					"email": "hello@example.com",
					"groups": [
						"Everyone",
						"Developers"
					]
				}`,
			verifyFunc: func(t *testing.T, info *InfoClaims, err error) {
				assert.NilError(t, err, "required_without")
				assert.Equal(t, info.Email, "hello@example.com")
				assert.Equal(t, info.Name, "hello")
				assert.Assert(t, len(info.Groups) == 2)

				parsedGroups := make(map[string]bool)
				for _, group := range info.Groups {
					parsedGroups[group] = true
				}

				assert.Assert(t, parsedGroups["Everyone"])
				assert.Assert(t, parsedGroups["Developers"])
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server, ctx := setupOIDCTest(t, test.infoResponse)
			serverURL := server.run(t, nil)
			provider := NewOIDC(models.Provider{Kind: models.OIDCKind, URL: serverURL, ClientID: "invalid"}, "invalid", "http://localhost:8301")
			info, err := provider.GetUserInfo(ctx, &models.ProviderUser{AccessToken: "aaa", RefreshToken: "bbb", ExpiresAt: time.Now().UTC().Add(5 * time.Minute)})
			test.verifyFunc(t, info, err)
		})
	}
}
