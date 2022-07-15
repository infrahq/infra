package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	gocmp "github.com/google/go-cmp/cmp"
	"github.com/prometheus/client_golang/prometheus"
	"gorm.io/gorm"
	"gotest.tools/v3/assert"

	"github.com/infrahq/infra/api"
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

func TestListKeys(t *testing.T) {
	db := setupDB(t)
	s := &Server{
		db: db,
	}
	handlers := &API{
		server: s,
	}

	user := &models.Identity{Model: models.Model{ID: uid.New()}, Name: "foo@example.com"}
	err := data.CreateIdentity(db, user)
	assert.NilError(t, err)
	provider := data.InfraProvider(db)
	err = data.CreateGrant(db, &models.Grant{
		Subject:   user.PolyID(),
		Privilege: "admin",
		Resource:  "infra",
	})
	assert.NilError(t, err)

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Set("db", db)
	c.Set("identity", user)

	_, err = data.CreateAccessKey(db, &models.AccessKey{
		Name:       "foo",
		IssuedFor:  user.ID,
		ProviderID: provider.ID,
		ExpiresAt:  time.Now().UTC().Add(5 * time.Minute),
	})
	assert.NilError(t, err)

	_, err = data.CreateAccessKey(db, &models.AccessKey{
		Name:       "expired",
		IssuedFor:  user.ID,
		ProviderID: provider.ID,
		ExpiresAt:  time.Now().UTC().Add(-5 * time.Minute),
	})
	assert.NilError(t, err)

	_, err = data.CreateAccessKey(db, &models.AccessKey{
		Name:              "not_extended",
		IssuedFor:         user.ID,
		ProviderID:        provider.ID,
		ExpiresAt:         time.Now().UTC().Add(5 * time.Minute),
		ExtensionDeadline: time.Now().UTC().Add(-5 * time.Minute),
	})
	assert.NilError(t, err)

	resp, err := handlers.ListAccessKeys(c, &api.ListAccessKeysRequest{})
	assert.NilError(t, err)

	assert.Assert(t, len(resp.Items) > 0)
	assert.Equal(t, resp.Count, len(resp.Items))
	assert.Equal(t, resp.Items[0].IssuedForName, user.Name)

	srv := setupServer(t, withAdminUser)
	routes := srv.GenerateRoutes(prometheus.NewRegistry())

	t.Run("expired", func(t *testing.T) {
		for _, item := range resp.Items {
			assert.Assert(t, item.Expires.Time().UTC().After(time.Now().UTC()) || item.Expires.Time().IsZero())
			assert.Assert(t, item.ExtensionDeadline.Time().UTC().After(time.Now().UTC()) || item.ExtensionDeadline.Time().IsZero())
		}

		notExpiredLength := len(resp.Items)
		resp, err = handlers.ListAccessKeys(c, &api.ListAccessKeysRequest{ShowExpired: true})
		assert.NilError(t, err)

		assert.Equal(t, notExpiredLength, len(resp.Items)-2) // test showExpired in request
	})

	t.Run("sort", func(t *testing.T) {
		sort.SliceIsSorted(resp.Items, func(i, j int) bool {
			return resp.Items[i].Name < resp.Items[j].Name
		})
	})

	t.Run("latest", func(t *testing.T) {
		resp := httptest.NewRecorder()
		req, err := http.NewRequest(http.MethodGet, "/api/access-keys", nil)
		assert.NilError(t, err)
		req.Header.Add("Authorization", "Bearer "+adminAccessKey(srv))
		req.Header.Add("Infra-Version", "0.12.3")

		routes.ServeHTTP(resp, req)
		assert.Equal(t, resp.Code, http.StatusOK)

		resp1 := &api.ListResponse[api.AccessKey]{}
		err = json.Unmarshal(resp.Body.Bytes(), resp1)
		assert.NilError(t, err)

		assert.Assert(t, len(resp1.Items) > 0)
	})

	t.Run("no version header", func(t *testing.T) {
		resp := httptest.NewRecorder()
		req, err := http.NewRequest(http.MethodGet, "/v1/access-keys", nil)
		assert.NilError(t, err)
		req.Header.Add("Authorization", "Bearer "+adminAccessKey(srv))

		routes.ServeHTTP(resp, req)
		assert.Equal(t, resp.Code, http.StatusBadRequest)

		errMsg := api.Error{}
		err = json.Unmarshal(resp.Body.Bytes(), &errMsg)
		assert.NilError(t, err)

		assert.Assert(t, strings.Contains(errMsg.Message, "Infra-Version header required"))
		assert.Equal(t, errMsg.Code, int32(400))
	})

	t.Run("old version upgrades", func(t *testing.T) {
		resp := httptest.NewRecorder()
		req, err := http.NewRequest(http.MethodGet, "/v1/access-keys", nil)
		assert.NilError(t, err)
		req.Header.Add("Authorization", "Bearer "+adminAccessKey(srv))
		req.Header.Add("Infra-Version", "0.12.2")

		routes.ServeHTTP(resp, req)
		assert.Equal(t, resp.Code, http.StatusOK)

		resp2 := []api.AccessKey{}
		err = json.Unmarshal(resp.Body.Bytes(), &resp2)
		t.Log(resp.Body.String())
		assert.NilError(t, err)

		assert.Assert(t, len(resp2) > 0)
	})
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

func runStep(t *testing.T, name string, fn func(t *testing.T)) {
	if !t.Run(name, fn) {
		t.FailNow()
	}
}

var cmpAPIUserJSON = gocmp.Options{
	gocmp.FilterPath(pathMapKey(`created`, `updated`, `lastSeenAt`), cmpApproximateTime),
	gocmp.FilterPath(pathMapKey(`id`), cmpAnyValidUID),
}

func TestAPI_CreateAccessKey(t *testing.T) {
	type testCase struct {
		name     string
		setup    func(t *testing.T) api.CreateAccessKeyRequest
		expected func(t *testing.T, response *httptest.ResponseRecorder)
	}

	srv := setupServer(t, withAdminUser)
	routes := srv.GenerateRoutes(prometheus.NewRegistry())

	userResp := createUser(t, srv, routes, "usera@example.com")

	run := func(t *testing.T, tc testCase) {
		body := tc.setup(t)

		req, err := http.NewRequest(http.MethodPost, "/api/access-keys", jsonBody(t, body))
		assert.NilError(t, err)
		req.Header.Add("Authorization", "Bearer "+adminAccessKey(srv))

		resp := httptest.NewRecorder()
		routes.ServeHTTP(resp, req)

		tc.expected(t, resp)
	}

	var testCases = []testCase{
		{
			name: "automatic name",
			setup: func(t *testing.T) api.CreateAccessKeyRequest {
				return api.CreateAccessKeyRequest{
					UserID:            userResp.ID,
					TTL:               api.Duration(time.Minute),
					ExtensionDeadline: api.Duration(time.Minute),
				}
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusCreated, resp.Body.String())

				respBody := &api.CreateAccessKeyResponse{}
				err := json.Unmarshal(resp.Body.Bytes(), respBody)
				assert.NilError(t, err)
				assert.Assert(t, strings.HasPrefix(respBody.Name, "usera@example.com-"), respBody.Name)
			},
		},
		{
			name: "user provided name",
			setup: func(t *testing.T) api.CreateAccessKeyRequest {
				return api.CreateAccessKeyRequest{
					UserID:            userResp.ID,
					Name:              "mysupersecretaccesskey",
					TTL:               api.Duration(time.Minute),
					ExtensionDeadline: api.Duration(time.Minute),
				}
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusCreated, resp.Body.String())

				respBody := &api.CreateAccessKeyResponse{}
				err := json.Unmarshal(resp.Body.Bytes(), respBody)
				assert.NilError(t, err)
				assert.Equal(t, respBody.Name, "mysupersecretaccesskey")
			},
		},
		{
			name: "invalid name",
			setup: func(t *testing.T) api.CreateAccessKeyRequest {
				return api.CreateAccessKeyRequest{
					UserID:            userResp.ID,
					Name:              "this-name-should-not-contain-slash/",
					TTL:               api.Duration(time.Minute),
					ExtensionDeadline: api.Duration(time.Minute),
				}
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusBadRequest, resp.Body.String())

				respBody := &api.Error{}
				err := json.Unmarshal(resp.Body.Bytes(), respBody)
				assert.NilError(t, err)

				expected := []api.FieldError{
					{FieldName: "name", Errors: []string{"character / at position 34 is not allowed"}},
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

func TestAPI_ListAccessKey(t *testing.T) {
	srv := setupServer(t, withAdminUser)
	routes := srv.GenerateRoutes(prometheus.NewRegistry())

	run := func() api.ListResponse[api.AccessKey] {
		req := httptest.NewRequest(http.MethodGet, "/api/access-keys", nil)
		req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", adminAccessKey(srv)))

		resp := httptest.NewRecorder()
		routes.ServeHTTP(resp, req)

		assert.Equal(t, resp.Code, http.StatusOK, resp.Body.String())

		var respBody api.ListResponse[api.AccessKey]
		err := json.Unmarshal(resp.Body.Bytes(), &respBody)
		assert.NilError(t, err)
		return respBody
	}

	t.Run("OK", func(t *testing.T) {
		accessKeys := run()
		// non-zero since there's an access key for the admin user
		assert.Assert(t, accessKeys.Count != 0)
		assert.Assert(t, accessKeys.Items != nil)
	})

	t.Run("MissingIssuedFor", func(t *testing.T) {
		err := srv.db.Create(&models.AccessKey{Name: "testing"}).Error
		assert.NilError(t, err)

		accessKeys := run()
		assert.Assert(t, accessKeys.Count != 0)
		assert.Assert(t, accessKeys.Items != nil)

		var accessKey *api.AccessKey
		for i := range accessKeys.Items {
			if accessKeys.Items[i].Name == "testing" {
				accessKey = &accessKeys.Items[i]
			}
		}

		assert.Assert(t, accessKey.Name == "testing")
		assert.Assert(t, accessKey.IssuedFor == 0)
		assert.Assert(t, accessKey.IssuedForName == "")
	})
}
