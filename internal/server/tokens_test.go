package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"gotest.tools/v3/assert"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
)

func TestAPI_CreateToken(t *testing.T) {
	srv := setupServer(t, withAdminUser)
	routes := srv.GenerateRoutes()

	type testCase struct {
		setup    func(t *testing.T, req *http.Request)
		expected func(t *testing.T, resp *httptest.ResponseRecorder)
	}

	run := func(t *testing.T, tc testCase) {
		// nolint:noctx
		req := httptest.NewRequest(http.MethodPost, "/api/tokens", nil)
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
		"success": {
			setup: func(t *testing.T, req *http.Request) {
				user := &models.Identity{
					Name: "spike@example.com",
				}
				err := data.CreateIdentity(srv.DB(), user)
				assert.NilError(t, err)
				_, err = data.CreateProviderUser(srv.DB(), data.InfraProvider(srv.DB()), user)
				assert.NilError(t, err)

				key := &models.AccessKey{
					IssuedForID: user.ID,
					ProviderID:  data.InfraProvider(srv.DB()).ID,
					ExpiresAt:   time.Now().Add(10 * time.Second),
				}
				accessKey, err := data.CreateAccessKey(srv.DB(), key)
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
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			run(t, tc)
		})
	}
}
