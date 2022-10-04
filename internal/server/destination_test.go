package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	gocmp "github.com/google/go-cmp/cmp"
	"gotest.tools/v3/assert"

	"github.com/infrahq/infra/api"
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
					{FieldName: "uniqueID", Errors: []string{"is required"}},
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
	gocmp.FilterPath(pathMapKey(`created`, `updated`), cmpApproximateTime),
	gocmp.FilterPath(pathMapKey(`id`), cmpAnyValidUID),
}
