package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

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

	db := setupDB(t)

	s := &Server{db: db}
	routes := s.GenerateRoutes()

	run := func(t *testing.T, tc testCase) {
		req, err := http.NewRequest(http.MethodGet, "/debug/pprof/heap?debug=1", nil)
		require.NoError(t, err)

		if tc.setupRequest != nil {
			tc.setupRequest(t, req)
		}

		resp := httptest.NewRecorder()
		routes.ServeHTTP(resp, req)

		require.Equal(t, tc.expectedCode, resp.Code, resp.Body.String())
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
				key, _ := createAccessKey(t, s.db, "user1@example.com")
				req.Header.Add("Authorization", "Bearer "+key)
			},
			expectedResp: responseBodyAPIErrorWithCode(http.StatusForbidden),
		},
		{
			name:         "successful profile",
			expectedCode: http.StatusOK,
			setupRequest: func(t *testing.T, req *http.Request) {
				key, user := createAccessKey(t, s.db, "user2@example.com")
				err := data.CreateGrant(s.db, &models.Grant{
					Subject:   user.PolyID(),
					Privilege: models.InfraAdminRole,
					Resource:  access.ResourceInfraAPI,
					CreatedBy: user.ID,
				})
				require.NoError(t, err)

				req.Header.Add("Authorization", "Bearer "+key)
			},
			expectedResp: func(t *testing.T, resp *httptest.ResponseRecorder) {
				require.Equal(t, "text/plain; charset=utf-8", resp.Header().Get("Content-Type"))
				require.Contains(t, resp.Body.String(), "heap profile:")
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
		require.NoError(t, err)
		require.Equal(t, apiError.Code, code)
	}
}

func createAccessKey(t *testing.T, db *gorm.DB, email string) (string, *models.Identity) {
	t.Helper()
	user := &models.Identity{Name: email, Kind: models.UserKind}
	err := data.CreateIdentity(db, user)
	require.NoError(t, err)

	token := &models.AccessKey{
		IssuedFor: user.ID,
		ExpiresAt: time.Now().Add(10 * time.Second),
	}

	body, err := data.CreateAccessKey(db, token)
	require.NoError(t, err)

	return body, user
}
