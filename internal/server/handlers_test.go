package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	gocmp "github.com/google/go-cmp/cmp"
	"gorm.io/gorm"
	"gotest.tools/v3/assert"

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

func createAdmin(t *testing.T, db *gorm.DB) *models.Identity {
	user := &models.Identity{
		Name: "admin+" + generate.MathRandom(10, generate.CharsetAlphaNumeric),
	}
	err := data.CreateIdentity(db, user)
	assert.NilError(t, err)

	err = data.CreateGrant(db, &models.Grant{
		Subject:   uid.NewIdentityPolymorphicID(user.ID),
		Resource:  models.InternalInfraProviderName,
		Privilege: models.InfraAdminRole,
	})
	assert.NilError(t, err)

	return user
}

func loginAs(db *gorm.DB, user *models.Identity) *gin.Context {
	ctx, _ := gin.CreateTestContext(nil)
	ctx.Set("db", db)
	ctx.Set("identity", user)
	return ctx
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

var cmpAPIUserJSON = gocmp.Options{
	gocmp.FilterPath(pathMapKey(`created`, `updated`, `lastSeenAt`), cmpApproximateTime),
	gocmp.FilterPath(pathMapKey(`id`), cmpAnyValidUID),
}

func TestWellKnownJWKs(t *testing.T) {
	srv := setupServer(t, withAdminUser, withSupportAdminGrant)
	routes := srv.GenerateRoutes()

	type testCase struct {
		name     string
		expected func(t *testing.T, resp *httptest.ResponseRecorder)
	}

	run := func(t *testing.T, tc testCase) {
		// nolint:noctx
		req, err := http.NewRequest(http.MethodGet, "/.well-known/jwks.json", nil)
		assert.NilError(t, err)

		resp := httptest.NewRecorder()
		routes.ServeHTTP(resp, req)

		tc.expected(t, resp)
	}

	testCases := []testCase{
		{
			name: "success",
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
		// TODO(orgs): add test case with other org
		// TODO(orgs): add test case with non-existent org
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
