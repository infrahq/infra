package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/Masterminds/semver/v3"
	gocmp "github.com/google/go-cmp/cmp"
	"github.com/prometheus/client_golang/prometheus"
	"gopkg.in/square/go-jose.v2"
	"gotest.tools/v3/assert"

	"github.com/infrahq/infra/internal/ginutil"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
)

func TestMain(m *testing.M) {
	// set mode so that test failure output is not filled by gin debug output by default
	ginutil.SetMode()
	os.Exit(m.Run())
}

func TestWellKnownJWKs(t *testing.T) {
	srv := setupServer(t, withAdminUser, withSupportAdminGrant)
	routes := srv.GenerateRoutes()
	srv.options.EnableSignup = true

	var defaultKey jose.JSONWebKey
	err := defaultKey.UnmarshalJSON(srv.db.DefaultOrg.PublicJWK)
	assert.NilError(t, err)

	otherOrg := &models.Organization{Name: "Other", Domain: "other.example.org"}
	createOrgs(t, srv.db, otherOrg)

	otherOrgTx := txnForTestCase(t, srv.db, otherOrg.ID)

	var otherOrgKey jose.JSONWebKey
	err = otherOrgKey.UnmarshalJSON(otherOrg.PublicJWK)
	assert.NilError(t, err)

	connector := data.InfraConnectorIdentity(otherOrgTx)
	accessKey, err := data.CreateAccessKey(otherOrgTx, &models.AccessKey{
		IssuedFor:  connector.ID,
		ExpiresAt:  time.Now().Add(20 * time.Second),
		ProviderID: data.InfraProvider(otherOrgTx).ID,
	})
	assert.NilError(t, err)

	assert.NilError(t, otherOrgTx.Commit())

	type testCase struct {
		name     string
		setup    func(t *testing.T, req *http.Request)
		expected func(t *testing.T, resp *httptest.ResponseRecorder)
	}

	run := func(t *testing.T, tc testCase) {
		// nolint:noctx
		req := httptest.NewRequest(http.MethodGet, "/.well-known/jwks.json", nil)

		if tc.setup != nil {
			tc.setup(t, req)
		}

		resp := httptest.NewRecorder()
		routes.ServeHTTP(resp, req)

		tc.expected(t, resp)
	}

	testCases := []testCase{
		{
			name: "default org with signup disabled",
			setup: func(t *testing.T, req *http.Request) {
				srv.options.EnableSignup = false
				t.Cleanup(func() {
					srv.options.EnableSignup = true
				})
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusOK, resp.Body.String())

				body := jsonUnmarshal(t, resp.Body.String())
				expected := map[string]any{
					"keys": []any{
						map[string]any{
							"alg": "ED25519",
							"crv": "Ed25519",
							"kty": "OKP",
							"use": "sig",
							"kid": "<any-string>",
							"x":   "<any-string>",
						},
					},
				}
				assert.DeepEqual(t, body, expected, cmpWellKnownJWKsJSON)
			},
		},
		{
			name: "default org",
			setup: func(t *testing.T, req *http.Request) {
				req.Header.Set("Authorization", "Bearer "+adminAccessKey(srv))
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusOK, resp.Body.String())

				var response WellKnownJWKResponse
				assert.NilError(t, json.NewDecoder(resp.Body).Decode(&response))
				assert.Equal(t, len(response.Keys), 1)

				assert.NilError(t, err)
				assert.DeepEqual(t, response.Keys[0], defaultKey)
			},
		},
		{
			name: "other org from access key",
			setup: func(t *testing.T, req *http.Request) {
				req.Header.Set("Authorization", "Bearer "+accessKey)
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusOK, resp.Body.String())

				var response WellKnownJWKResponse
				assert.NilError(t, json.NewDecoder(resp.Body).Decode(&response))
				assert.Equal(t, len(response.Keys), 1)

				assert.NilError(t, err)
				assert.DeepEqual(t, response.Keys[0], otherOrgKey)
			},
		},
		{
			name: "unknown org",
			setup: func(t *testing.T, req *http.Request) {
				req.Host = "something.unknown.example.org"
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusBadRequest, resp.Body.String())
			},
		},
		{
			name: "org from domain name",
			setup: func(t *testing.T, req *http.Request) {
				req.Host = "other.example.org"
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusOK, resp.Body.String())

				var response WellKnownJWKResponse
				assert.NilError(t, json.NewDecoder(resp.Body).Decode(&response))
				assert.Equal(t, len(response.Keys), 1)

				assert.NilError(t, err)
				assert.DeepEqual(t, response.Keys[0], otherOrgKey)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			run(t, tc)
		})
	}
}

var cmpWellKnownJWKsJSON = gocmp.Options{
	gocmp.FilterPath(pathMapKey(`kid`, `x`), cmpAnyString),
}

func TestAPI_AddPreviousVersionHandlers_Order(t *testing.T) {
	srv := newServer(Options{})
	srv.metricsRegistry = prometheus.NewRegistry()
	routes := srv.GenerateRoutes()

	for routeID, versions := range routes.api.versions {
		prev := semver.MustParse("0.0.0")
		for _, v := range versions {
			assert.Assert(t, prev.LessThan(v.version),
				"handlers for %v are in the wrong order, %v should be before %v",
				routeID, v.version, prev)
			prev = v.version
		}
	}
}
