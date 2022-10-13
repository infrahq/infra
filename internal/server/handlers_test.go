package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	gocmp "github.com/google/go-cmp/cmp"
	"gopkg.in/square/go-jose.v2"
	"gotest.tools/v3/assert"

	"github.com/infrahq/infra/internal/access"
	"github.com/infrahq/infra/internal/generate"
	"github.com/infrahq/infra/internal/ginutil"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

func TestMain(m *testing.M) {
	// set mode so that test failure output is not filled by gin debug output by default
	ginutil.SetMode()
	os.Exit(m.Run())
}

func adminAccessKey(s *Server) string {
	for _, id := range s.options.Users {
		if id.Name == "admin@example.com" {
			return id.AccessKey
		}
	}

	return ""
}

// withAdminUser may be used with setupServer to setup the server
// with an admin identity and access key
func withAdminUser(_ *testing.T, opts *Options) {
	opts.Users = append(opts.Users, User{
		Name:      "admin@example.com",
		AccessKey: "BlgpvURSGF.NdcemBdzxLTGIcjPXwPoZNrb",
	})
	opts.Grants = append(opts.Grants, Grant{
		User:     "admin@example.com",
		Role:     "admin",
		Resource: "infra",
	})
}

func withSupportAdminGrant(_ *testing.T, opts *Options) {
	opts.Grants = append(opts.Grants, Grant{
		User:     "admin@example.com",
		Role:     "support-admin",
		Resource: "infra",
	})
}

func withMultiOrgEnabled(_ *testing.T, opts *Options) {
	opts.DefaultOrganizationDomain = "example.com"
	opts.EnableSignup = true
}

func createAdmin(t *testing.T, db data.GormTxn) *models.Identity {
	user := &models.Identity{
		Name: "admin+" + generate.MathRandom(10, generate.CharsetAlphaNumeric),
	}
	err := data.CreateIdentity(db, user)
	assert.NilError(t, err)

	err = data.CreateGrant(db, &models.Grant{
		Subject:   uid.NewIdentityPolymorphicID(user.ID),
		Resource:  "infra",
		Privilege: models.InfraAdminRole,
	})
	assert.NilError(t, err)

	return user
}

func loginAs(tx *data.Transaction, user *models.Identity) *gin.Context {
	ctx, _ := gin.CreateTestContext(nil)
	ctx.Set(access.RequestContextKey, access.RequestContext{
		DBTxn:         tx,
		Authenticated: access.Authenticated{User: user},
	})
	return ctx
}

func txnForTestCase(t *testing.T, db *data.DB, orgID uid.ID) *data.Transaction {
	t.Helper()
	tx, err := db.Begin(context.Background(), nil)
	assert.NilError(t, err)
	t.Cleanup(func() {
		assert.NilError(t, tx.Rollback())
	})
	return tx.WithMetadata(orgID)
}

func jsonBody(t *testing.T, body interface{}) *bytes.Buffer {
	t.Helper()
	buf := new(bytes.Buffer)
	err := json.NewEncoder(buf).Encode(body)
	assert.NilError(t, err)
	return buf
}

// cmpApproximateTime is a gocmp.Option that compares a time formatted as an
// RFC3339 string. The times may be up to 2 seconds different from each other,
// to account for the runtime of a test.
// cmpApproximateTime accepts interface{} instead of time.Time because it is
// intended to be used to compare times in API responses that were decoded
// into an interface{}.
var cmpApproximateTime = gocmp.Comparer(func(x, y interface{}) bool {
	xs, _ := x.(string)
	xd, _ := time.Parse(time.RFC3339, xs)

	ys, _ := y.(string)
	yd, _ := time.Parse(time.RFC3339, ys)

	if xd.After(yd) {
		xd, yd = yd, xd
	}
	return yd.Sub(xd) < 30*time.Second
})

// cmpAnyValidUID is a gocmp.Option that allows a field to match any valid uid.ID,
// as long as the expected value is the literal string "<any-valid-uid>".
// cmpAnyValidUID accepts interface{} instead of string because it is intended
// to be used to compare a UID.ID in API responses that were decoded
// into an interface{}.
var cmpAnyValidUID = gocmp.Comparer(func(x, y interface{}) bool {
	xs, _ := x.(string)
	ys, _ := y.(string)

	if xs == "<any-valid-uid>" {
		_, err := uid.Parse([]byte(ys))
		return err == nil
	}
	if ys == "<any-valid-uid>" {
		_, err := uid.Parse([]byte(xs))
		return err == nil
	}
	return xs == ys
})

// pathMapKey is a gocmp.FilerPath filter that matches map entries with any
// of the keys.
// TODO: allow dotted identifier for keys in nested maps.
func pathMapKey(keys ...string) func(path gocmp.Path) bool {
	return func(path gocmp.Path) bool {
		mapIndex, ok := path.Last().(gocmp.MapIndex)
		if !ok {
			return false
		}

		for _, key := range keys {
			if mapIndex.Key().Interface() == key {
				return true
			}
		}
		return false
	}
}

func jsonUnmarshal(t *testing.T, raw string) interface{} {
	t.Helper()
	var out interface{}
	err := json.Unmarshal([]byte(raw), &out)
	assert.NilError(t, err, "failed to decode JSON")
	return out
}

var cmpAPIOrganizationJSON = gocmp.Options{
	gocmp.FilterPath(pathMapKey(`created`, `updated`), cmpApproximateTime),
}

var cmpAPIUserJSON = gocmp.Options{
	gocmp.FilterPath(pathMapKey(`created`, `updated`, `lastSeenAt`), cmpApproximateTime),
	gocmp.FilterPath(pathMapKey(`id`), cmpAnyValidUID),
}

type organizationData struct {
	Organization   *models.Organization
	Admin          *models.Identity
	AdminAccessKey string
}

// createOtherOrg creates an organization with domain other.example.org, with
// a user, and a grant that makes them an infra admin. It can be used by tests
// to ensure that an API endpoint honors the organization of the user making
// the request.
func createOtherOrg(t *testing.T, db *data.DB) organizationData {
	t.Helper()
	otherOrg := &models.Organization{Name: "Other", Domain: "other.example.org"}
	assert.NilError(t, data.CreateOrganization(db, otherOrg))

	tx := txnForTestCase(t, db, otherOrg.ID)
	admin := createAdmin(t, tx)

	token := &models.AccessKey{
		IssuedFor:  admin.ID,
		ProviderID: data.InfraProvider(tx).ID,
		ExpiresAt:  time.Now().Add(1000 * time.Second),
	}

	accessKey, err := data.CreateAccessKey(tx, token)
	assert.NilError(t, err)

	assert.NilError(t, tx.Commit())
	return organizationData{
		Organization:   otherOrg,
		Admin:          admin,
		AdminAccessKey: accessKey,
	}
}

func TestWellKnownJWKs(t *testing.T) {
	srv := setupServer(t, withAdminUser, withSupportAdminGrant)
	routes := srv.GenerateRoutes()
	srv.options.EnableSignup = true

	otherOrg := &models.Organization{Name: "Other", Domain: "other.example.org"}
	createOrgs(t, srv.db, otherOrg)

	settings, err := data.GetSettings(srv.db)
	assert.NilError(t, err)

	var defaultKey jose.JSONWebKey
	err = defaultKey.UnmarshalJSON(settings.PublicJWK)
	assert.NilError(t, err)

	otherOrgTx := txnForTestCase(t, srv.db, otherOrg.ID)
	settings, err = data.GetSettings(otherOrgTx)
	assert.NilError(t, err)

	var otherOrgKey jose.JSONWebKey
	err = otherOrgKey.UnmarshalJSON(settings.PublicJWK)
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
		req, err := http.NewRequest(http.MethodGet, "/.well-known/jwks.json", nil)
		assert.NilError(t, err)

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

// cmpAnyString is a gocmp.Option that allows a field to match any non-zero string.
var cmpAnyString = gocmp.Comparer(func(x, y interface{}) bool {
	xs, _ := x.(string)
	ys, _ := y.(string)

	if xs == "" || ys == "" {
		return false
	}
	if xs == "<any-string>" || ys == "<any-string>" {
		return true
	}
	return xs == ys
})
