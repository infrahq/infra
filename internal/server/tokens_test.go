package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"gotest.tools/v3/assert"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal/access"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/internal/server/providers"
)

func TestAPI_CreateToken(t *testing.T) {
	srv := setupServer(t, withAdminUser)
	routes := srv.GenerateRoutes(prometheus.NewRegistry())

	type testCase struct {
		setup    func(t *testing.T, req *http.Request)
		expected func(t *testing.T, resp *httptest.ResponseRecorder)
	}

	run := func(t *testing.T, tc testCase) {
		req, err := http.NewRequest(http.MethodPost, "/api/tokens", nil)
		assert.NilError(t, err)
		req.Header.Add("Infra-Version", apiVersionLatest)

		if tc.setup != nil {
			tc.setup(t, req)
		}

		resp := httptest.NewRecorder()
		routes.ServeHTTP(resp, req)

		tc.expected(t, resp)
	}

	testCases := map[string]testCase{
		"not authenticated": {
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusUnauthorized)
			},
		},
		"infra provider user with valid access key": {
			setup: func(t *testing.T, req *http.Request) {
				user := &models.Identity{
					Name: "spike@example.com",
				}
				err := data.CreateIdentity(srv.db, user)
				assert.NilError(t, err)
				_, err = data.CreateProviderUser(srv.db, data.InfraProvider(srv.db), user)
				assert.NilError(t, err)

				key := &models.AccessKey{
					IssuedFor:  user.ID,
					ProviderID: data.InfraProvider(srv.db).ID,
					ExpiresAt:  time.Now().Add(10 * time.Second),
				}
				accessKey, err := data.CreateAccessKey(srv.db, key)
				assert.NilError(t, err)

				req.Header.Set("Authorization", "Bearer "+accessKey)
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusCreated)

				respBody := &api.CreateTokenResponse{}
				err := json.Unmarshal(resp.Body.Bytes(), respBody)
				assert.NilError(t, err)
				assert.Assert(t, respBody.Token != "")
			},
		},
		"access key directly created for user not in infra provider": {
			setup: func(t *testing.T, req *http.Request) {
				user := &models.Identity{
					Name: "faye@example.com",
				}
				err := data.CreateIdentity(srv.db, user)
				assert.NilError(t, err)

				key := &models.AccessKey{
					IssuedFor:  user.ID,
					ProviderID: data.InfraProvider(srv.db).ID,
					ExpiresAt:  time.Now().Add(10 * time.Second),
				}
				accessKey, err := data.CreateAccessKey(srv.db, key)
				assert.NilError(t, err)

				req.Header.Set("Authorization", "Bearer "+accessKey)
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusCreated)
			},
		},
		"access key for valid idp user": {
			setup: func(t *testing.T, req *http.Request) {
				user := &models.Identity{
					Name: "jet@example.com",
				}
				err := data.CreateIdentity(srv.db, user)
				assert.NilError(t, err)

				provider := &models.Provider{
					Name: "mockta",
					Kind: models.ProviderKindOIDC,
				}
				err = data.CreateProvider(srv.db, provider)
				assert.NilError(t, err)

				_, err = data.CreateProviderUser(srv.db, provider, user)
				assert.NilError(t, err)

				key := &models.AccessKey{
					IssuedFor:  user.ID,
					ProviderID: provider.ID,
					ExpiresAt:  time.Now().Add(10 * time.Second),
				}
				accessKey, err := data.CreateAccessKey(srv.db, key)
				assert.NilError(t, err)

				ctx := providers.WithOIDCClient(req.Context(), &fakeOIDCImplementation{})
				rCtx := access.RequestContext{
					Request: req,
					DBTxn:   srv.db,
					Authenticated: access.Authenticated{
						AccessKey: key,
						User:      user,
					},
				}

				// nolint: staticcheck
				ctx = context.WithValue(ctx, access.RequestContextKey, rCtx)

				*req = *req.WithContext(ctx)

				req.Header.Set("Authorization", "Bearer "+accessKey)
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusCreated)

				respBody := &api.CreateTokenResponse{}
				err := json.Unmarshal(resp.Body.Bytes(), respBody)
				assert.NilError(t, err)
				assert.Assert(t, respBody.Token != "")
			},
		},
		"access key for revoked idp user": {
			setup: func(t *testing.T, req *http.Request) {
				user := &models.Identity{
					Name: "ein@example.com",
				}
				err := data.CreateIdentity(srv.db, user)
				assert.NilError(t, err)

				provider := &models.Provider{
					Name: "mockta-revoked-user",
					Kind: models.ProviderKindOIDC,
				}
				err = data.CreateProvider(srv.db, provider)
				assert.NilError(t, err)

				_, err = data.CreateProviderUser(srv.db, provider, user)
				assert.NilError(t, err)

				key := &models.AccessKey{
					IssuedFor:  user.ID,
					ProviderID: provider.ID,
					ExpiresAt:  time.Now().Add(10 * time.Second),
				}
				accessKey, err := data.CreateAccessKey(srv.db, key)
				assert.NilError(t, err)

				ctx := providers.WithOIDCClient(req.Context(), &fakeOIDCImplementation{UserInfoRevoked: true})
				rCtx := access.RequestContext{
					Request: req,
					DBTxn:   srv.db,
					Authenticated: access.Authenticated{
						AccessKey: key,
						User:      user,
					},
				}

				// nolint: staticcheck
				ctx = context.WithValue(ctx, access.RequestContextKey, rCtx)

				*req = *req.WithContext(ctx)

				req.Header.Set("Authorization", "Bearer "+accessKey)
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusUnauthorized)
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			run(t, tc)
		})
	}
}
