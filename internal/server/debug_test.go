package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal/access"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
)

func TestAPI_PProfHandler(t *testing.T) {
	type testCase struct {
		name         string
		setupRequest func(t *testing.T, req *http.Request)
		expectedCode int
		expectedResp func(t *testing.T, resp *httptest.ResponseRecorder)
	}

	s := setupServer(t)
	routes := s.GenerateRoutes()

	run := func(t *testing.T, tc testCase) {
		// nolint:noctx
		req := httptest.NewRequest(http.MethodGet, "/api/debug/pprof/heap?debug=1", nil)
		req.Header.Add("Infra-Version", "0.12.3")

		if tc.setupRequest != nil {
			tc.setupRequest(t, req)
		}

		resp := httptest.NewRecorder()
		routes.ServeHTTP(resp, req)

		assert.Equal(t, tc.expectedCode, resp.Code, resp.Body.String())
		if tc.expectedResp != nil {
			tc.expectedResp(t, resp)
		}
	}

	testCases := []testCase{
		{
			name:         "missing access key",
			expectedCode: http.StatusUnauthorized,
			expectedResp: responseBodyAPIErrorWithCode(http.StatusUnauthorized),
		},
		{
			name:         "missing admin role",
			expectedCode: http.StatusForbidden,
			setupRequest: func(_ *testing.T, req *http.Request) {
				key, _ := createAccessKey(t, s.DB(), "user1@example.com")
				req.Header.Add("Authorization", "Bearer "+key)
			},
			expectedResp: responseBodyAPIErrorWithCode(http.StatusForbidden),
		},
		{
			name:         "successful profile",
			expectedCode: http.StatusOK,
			setupRequest: func(t *testing.T, req *http.Request) {
				key, user := createAccessKey(t, s.DB(), "user2@example.com")
				err := data.CreateGrant(s.DB(), &models.Grant{
					Subject:   models.NewSubjectForUser(user.ID),
					Privilege: models.InfraSupportAdminRole,
					Resource:  access.ResourceInfraAPI,
					CreatedBy: user.ID,
				})
				assert.NilError(t, err)

				req.Header.Add("Authorization", "Bearer "+key)
			},
			expectedResp: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, "text/plain; charset=utf-8", resp.Header().Get("Content-Type"))
				assert.Assert(t, is.Contains(resp.Body.String(), "heap profile:"))
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			run(t, tc)
		})
	}
}

func responseBodyAPIErrorWithCode(code int32) func(t *testing.T, resp *httptest.ResponseRecorder) {
	return func(t *testing.T, resp *httptest.ResponseRecorder) {
		t.Helper()

		var apiError api.Error

		err := json.Unmarshal(resp.Body.Bytes(), &apiError)
		assert.NilError(t, err)
		assert.Equal(t, apiError.Code, code)
	}
}
