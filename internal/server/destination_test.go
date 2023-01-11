package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	gocmp "github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"gotest.tools/v3/assert"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
)

func TestAPI_CreateDestination(t *testing.T) {
	srv := setupServer(t, withAdminUser)
	routes := srv.GenerateRoutes()

	type testCase struct {
		name     string
		setup    func(t *testing.T) api.CreateDestinationRequest
		expected func(t *testing.T, resp *httptest.ResponseRecorder)
	}

	run := func(t *testing.T, tc testCase) {
		createReq := tc.setup(t)
		body := jsonBody(t, &createReq)
		req := httptest.NewRequest(http.MethodPost, "/api/destinations", body)
		req.Header.Set("Authorization", "Bearer "+adminAccessKey(srv))
		req.Header.Set("Infra-Version", apiVersionLatest)

		resp := httptest.NewRecorder()
		routes.ServeHTTP(resp, req)

		tc.expected(t, resp)
	}

	testCases := []testCase{
		{
			name: "does not trim trailing newline from CA",
			setup: func(t *testing.T) api.CreateDestinationRequest {
				return api.CreateDestinationRequest{
					Name:     "final",
					UniqueID: "unique-id",
					Connection: api.DestinationConnection{
						URL: "cluster.production.example",
						CA:  "-----BEGIN CERTIFICATE-----\nok\n-----END CERTIFICATE-----\n",
					},
					Resources: []string{"res1", "res2"},
					Roles:     []string{"role1", "role2"},
				}
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusCreated, resp.Body.String())

				expected := jsonUnmarshal(t, fmt.Sprintf(`
{
	"id": "<any-valid-uid>",
	"name": "final",
	"kind": "kubernetes",
	"uniqueID": "unique-id",
	"version": "",
	"connection": {
		"url": "cluster.production.example",
		"ca": "-----BEGIN CERTIFICATE-----\nok\n-----END CERTIFICATE-----\n"
	},
	"connected": false,
	"lastSeen": null,
	"resources": ["res1", "res2"],
	"roles": ["role1", "role2"],
	"created": "%[1]v",
	"updated": "%[1]v"
}
`,
					time.Now().UTC().Format(time.RFC3339)))

				actual := jsonUnmarshal(t, resp.Body.String())
				assert.DeepEqual(t, actual, expected, cmpAPIDestinationJSON)
			},
		},
		{
			name: "missing required fields",
			setup: func(t *testing.T) api.CreateDestinationRequest {
				return api.CreateDestinationRequest{}
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusBadRequest, resp.Body.String())

				respBody := &api.Error{}
				err := json.Unmarshal(resp.Body.Bytes(), respBody)
				assert.NilError(t, err)

				expected := []api.FieldError{
					{FieldName: "connection.ca", Errors: []string{"is required"}},
					{FieldName: "name", Errors: []string{"is required"}},
				}
				assert.DeepEqual(t, respBody.FieldErrors, expected)
			},
		},
		{
			name: "failed with reserved names",
			setup: func(t *testing.T) api.CreateDestinationRequest {
				return api.CreateDestinationRequest{
					Name: "infra",
					Connection: api.DestinationConnection{
						URL: "cluster.production.example",
						CA:  "the-ca",
					},
				}
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusBadRequest, resp.Body.String())

				respBody := &api.Error{}
				err := json.Unmarshal(resp.Body.Bytes(), respBody)
				assert.NilError(t, err)

				expected := []api.FieldError{
					{FieldName: "name", Errors: []string{"infra is reserved and can not be used"}},
				}
				assert.DeepEqual(t, respBody.FieldErrors, expected)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			run(t, tc)
		})
	}
}

var cmpAPIDestinationJSON = gocmp.Options{
	gocmp.FilterPath(pathMapKey(`created`, `updated`, `lastSeen`), cmpApproximateTime),
	gocmp.FilterPath(pathMapKey(`id`), cmpAnyValidUID),
}

func TestAPI_DeleteDestination(t *testing.T) {
	srv := setupServer(t, withAdminUser)
	routes := srv.GenerateRoutes()

	dest := &models.Destination{
		Name:     "wow",
		Kind:     models.DestinationKindKubernetes,
		UniqueID: "deadbeef",
	}
	assert.NilError(t, data.CreateDestination(srv.db, dest))

	type testCase struct {
		name     string
		setup    func(t *testing.T, req *http.Request)
		expected func(t *testing.T, resp *httptest.ResponseRecorder)
	}

	run := func(t *testing.T, tc testCase) {
		req := httptest.NewRequest(http.MethodDelete, "/api/destinations/"+dest.ID.String(), nil)
		req.Header.Set("Authorization", "Bearer "+adminAccessKey(srv))
		req.Header.Set("Infra-Version", apiVersionLatest)

		if tc.setup != nil {
			tc.setup(t, req)
		}

		resp := httptest.NewRecorder()
		routes.ServeHTTP(resp, req)

		tc.expected(t, resp)
	}

	testCases := []testCase{
		{
			name: "not authenticated",
			setup: func(t *testing.T, req *http.Request) {
				req.Header.Del("Authorization")
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusUnauthorized, (*responseDebug)(resp))
			},
		},
		{
			name: "not authorized",
			setup: func(t *testing.T, req *http.Request) {
				token, _ := createAccessKey(t, srv.db, "notauth@example.com")
				req.Header.Set("Authorization", "Bearer "+token)
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusForbidden, (*responseDebug)(resp))
			},
		},
		{
			name: "grants w/ more destinations",
			setup: func(t *testing.T, req *http.Request) {
				grants := []*models.Grant{
					{
						Subject:   "i:7654321",
						Privilege: "view",
						Resource:  "wow",
					},
					{
						Subject:   "i:7654321",
						Privilege: "view",
						Resource:  "wow.awesome",
					},
					{
						Subject:   "i:7654321",
						Privilege: "view",
						Resource:  "anotherthing",
					},
				}
				for _, g := range grants {
					assert.NilError(t, data.CreateGrant(srv.db, g))
				}
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusNoContent, (*responseDebug)(resp))

				grants, err := data.ListGrants(srv.db, data.ListGrantsOptions{BySubject: "i:7654321"})
				assert.NilError(t, err)
				assert.Equal(t, len(grants), 1)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			run(t, tc)
		})
	}

}

func TestAPI_UpdateDestination(t *testing.T) {
	srv := setupServer(t, withAdminUser)
	routes := srv.GenerateRoutes()

	dest := &models.Destination{
		Name:     "the-dest",
		Kind:     models.DestinationKindSSH,
		UniqueID: "unique-id",
	}
	assert.NilError(t, data.CreateDestination(srv.db, dest))

	type testCase struct {
		name     string
		setup    func(t *testing.T, req *http.Request)
		body     func(t *testing.T) api.UpdateDestinationRequest
		expected func(t *testing.T, resp *httptest.ResponseRecorder)
	}

	run := func(t *testing.T, tc testCase) {
		createReq := tc.body(t)
		body := jsonBody(t, &createReq)
		req := httptest.NewRequest(http.MethodPut, "/api/destinations/"+dest.ID.String(), body)
		req.Header.Set("Authorization", "Bearer "+adminAccessKey(srv))
		req.Header.Set("Infra-Version", apiVersionLatest)

		if tc.setup != nil {
			tc.setup(t, req)
		}

		resp := httptest.NewRecorder()
		routes.ServeHTTP(resp, req)

		tc.expected(t, resp)
	}

	testCases := []testCase{
		{
			name: "not authenticated",
			body: func(t *testing.T) api.UpdateDestinationRequest {
				return api.UpdateDestinationRequest{}
			},
			setup: func(t *testing.T, req *http.Request) {
				req.Header.Del("Authorization")
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusUnauthorized, (*responseDebug)(resp))
			},
		},
		{
			name: "not authorized",
			body: func(t *testing.T) api.UpdateDestinationRequest {
				return api.UpdateDestinationRequest{
					Name:     "the-dest",
					UniqueID: "unique-id",
					Connection: api.DestinationConnection{
						URL: "10.10.10.10:12345",
						CA:  "the-ca-or-fingerprint",
					},
				}
			},
			setup: func(t *testing.T, req *http.Request) {
				token, _ := createAccessKey(t, srv.db, "notauth@example.com")
				req.Header.Set("Authorization", "Bearer "+token)
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusForbidden, (*responseDebug)(resp))
			},
		},
		{
			name: "missing required fields",
			body: func(t *testing.T) api.UpdateDestinationRequest {
				return api.UpdateDestinationRequest{}
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusBadRequest, (*responseDebug)(resp))

				respBody := &api.Error{}
				err := json.Unmarshal(resp.Body.Bytes(), respBody)
				assert.NilError(t, err)

				expected := []api.FieldError{
					{FieldName: "connection.ca", Errors: []string{"is required"}},
					{FieldName: "name", Errors: []string{"is required"}},
				}
				assert.DeepEqual(t, respBody.FieldErrors, expected)
			},
		},
		{
			name: "success",
			body: func(t *testing.T) api.UpdateDestinationRequest {
				return api.UpdateDestinationRequest{
					Name:     "the-dest",
					UniqueID: "unique-id",
					Connection: api.DestinationConnection{
						URL: "10.10.10.10:12345",
						CA:  "the-ca-or-fingerprint",
					},
					Roles: []string{"one", "two"},
				}
			},
			setup: func(t *testing.T, req *http.Request) {
				// Set the header that connectors use
				req.Header.Set(headerInfraDestinationName, "the-dest")
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusOK, (*responseDebug)(resp))

				expectedBody := jsonUnmarshal(t, fmt.Sprintf(`
					{
						"id": "%[2]v",
						"name": "the-dest",
						"kind": "ssh",
						"uniqueID": "unique-id",
						"version": "",
						"connection": {
							"url": "10.10.10.10:12345",
							"ca": "the-ca-or-fingerprint"
						},
						"connected": true,
						"lastSeen": "%[1]v",
						"resources": null,
						"roles": ["one", "two"],
						"created": "%[1]v",
						"updated": "%[1]v"
					}
				`, time.Now().UTC().Format(time.RFC3339), dest.ID))

				actualBody := jsonUnmarshal(t, resp.Body.String())
				assert.DeepEqual(t, actualBody, expectedBody, cmpAPIDestinationJSON)

				expected := &models.Destination{
					Model: dest.Model,
					OrganizationMember: models.OrganizationMember{
						OrganizationID: srv.db.DefaultOrg.ID,
					},
					Name:          "the-dest",
					UniqueID:      "unique-id",
					Kind:          models.DestinationKindSSH,
					ConnectionURL: "10.10.10.10:12345",
					ConnectionCA:  "the-ca-or-fingerprint",
					LastSeenAt:    time.Now(),
					Roles:         []string{"one", "two"},
				}

				actual, err := data.GetDestination(srv.db, data.GetDestinationOptions{ByID: dest.ID})
				assert.NilError(t, err)

				var cmpDestination = gocmp.Options{
					cmpopts.EquateApproxTime(2 * time.Second),
					cmpopts.EquateEmpty(),
				}
				assert.DeepEqual(t, actual, expected, cmpDestination)
				assert.Assert(t, dest.UpdatedAt != actual.UpdatedAt)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			run(t, tc)
		})
	}
}
