package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	gocmp "github.com/google/go-cmp/cmp"
	"github.com/prometheus/client_golang/prometheus"
	"gotest.tools/v3/assert"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

func adminAccessKey(s *Server) string {
	for _, id := range s.options.Identities {
		if id.Name == "admin" {
			return id.AccessKey
		}
	}

	return ""
}

func TestAPI_ListIdentities(t *testing.T) {
	srv := setupServer(t, withAdminIdentity)
	routes := srv.GenerateRoutes(prometheus.NewRegistry())

	createID := func(t *testing.T, name string, kind string) uid.ID {
		t.Helper()
		var buf bytes.Buffer
		body := api.CreateIdentityRequest{Name: name, Kind: kind}
		err := json.NewEncoder(&buf).Encode(body)
		assert.NilError(t, err)

		req, err := http.NewRequest(http.MethodPost, "/v1/identities", &buf)
		assert.NilError(t, err)
		req.Header.Add("Authorization", "Bearer "+adminAccessKey(srv))

		resp := httptest.NewRecorder()
		routes.ServeHTTP(resp, req)
		assert.Equal(t, resp.Code, http.StatusCreated, resp.Body.String())
		respObj := &api.CreateIdentityResponse{}
		err = json.Unmarshal(resp.Body.Bytes(), respObj)
		assert.NilError(t, err)
		return respObj.ID
	}
	id1 := createID(t, "me@example.com", "user")
	id2 := createID(t, "other@example.com", "user")
	id3 := createID(t, "HAL", "machine")
	_ = createID(t, "other-HAL", "machine")

	type testCase struct {
		urlPath  string
		setup    func(t *testing.T, req *http.Request)
		expected func(t *testing.T, resp *httptest.ResponseRecorder)
	}

	run := func(t *testing.T, tc testCase) {
		req, err := http.NewRequest(http.MethodGet, tc.urlPath, nil)
		assert.NilError(t, err)
		req.Header.Add("Authorization", "Bearer "+adminAccessKey(srv))

		if tc.setup != nil {
			tc.setup(t, req)
		}

		resp := httptest.NewRecorder()
		routes.ServeHTTP(resp, req)

		tc.expected(t, resp)
	}

	testCases := map[string]testCase{
		"no name match": {
			urlPath: "/v1/identities?name=doesnotmatch",
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusOK)
				assert.Equal(t, resp.Body.String(), `[]`)
			},
		},
		"name match": {
			urlPath: "/v1/identities?name=me@example.com",
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusOK)

				var actual []api.Identity
				err := json.NewDecoder(resp.Body).Decode(&actual)
				assert.NilError(t, err)
				expected := []api.Identity{
					{Name: "me@example.com", Kind: "user"},
				}
				assert.DeepEqual(t, actual, expected, cmpAPIIdentityShallow)
			},
		},
		"filter by ids": {
			urlPath: fmt.Sprintf("/v1/identities?ids=%s&ids=%s&ids=%s", id1, id2, id3),
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusOK)

				var actual []api.Identity
				err := json.NewDecoder(resp.Body).Decode(&actual)
				assert.NilError(t, err)
				expected := []api.Identity{
					{Name: "HAL", Kind: "machine"},
					{Name: "me@example.com", Kind: "user"},
					{Name: "other@example.com", Kind: "user"},
				}
				assert.DeepEqual(t, actual, expected, cmpAPIIdentityShallow)
			},
		},
		"no filter": {
			urlPath: "/v1/identities",
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusOK)

				var actual []api.Identity
				err := json.NewDecoder(resp.Body).Decode(&actual)
				assert.NilError(t, err)
				expected := []api.Identity{
					{Name: "HAL", Kind: "machine"},
					{Name: "admin", Kind: "machine"},
					{Name: "connector", Kind: "machine"},
					{Name: "me@example.com", Kind: "user"},
					{Name: "other-HAL", Kind: "machine"},
					{Name: "other@example.com", Kind: "user"},
				}
				assert.DeepEqual(t, actual, expected, cmpAPIIdentityShallow)
			},
		},
		"no authorization": {
			urlPath: "/v1/identities",
			setup: func(t *testing.T, req *http.Request) {
				req.Header.Del("Authorization")
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusUnauthorized)
			},
		},
		// TODO: assert full JSON response
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			run(t, tc)
		})
	}
}

var cmpAPIIdentityShallow = gocmp.Comparer(func(x, y api.Identity) bool {
	return x.Name == y.Name && x.Kind == y.Kind
})

func TestListKeys(t *testing.T) {
	db := setupDB(t)
	s := &Server{
		db: db,
	}
	handlers := &API{
		server: s,
	}

	user := &models.Identity{Model: models.Model{ID: uid.New()}, Name: "foo@example.com", Kind: "user"}
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
		ExpiresAt:  time.Now().Add(5 * time.Minute),
	})
	assert.NilError(t, err)

	keys, err := handlers.ListAccessKeys(c, &api.ListAccessKeysRequest{})
	assert.NilError(t, err)

	assert.Assert(t, len(keys) > 0)

	assert.Equal(t, keys[0].IssuedForName, "foo@example.com")
}

func TestListProviders(t *testing.T) {
	s := setupServer(t, withAdminIdentity)
	routes := s.GenerateRoutes(prometheus.NewRegistry())

	testProvider := &models.Provider{Name: "mokta"}

	err := data.CreateProvider(s.db, testProvider)
	assert.NilError(t, err)

	dbProviders, err := data.ListProviders(s.db)
	assert.NilError(t, err)
	assert.Equal(t, len(dbProviders), 2)

	req, err := http.NewRequest(http.MethodGet, "/v1/providers", nil)
	assert.NilError(t, err)

	req.Header.Add("Authorization", "Bearer "+adminAccessKey(s))

	resp := httptest.NewRecorder()
	routes.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code, resp.Body.String())

	apiProviders := make([]api.Provider, 0)
	err = json.Unmarshal(resp.Body.Bytes(), &apiProviders)
	assert.NilError(t, err)

	assert.Equal(t, len(apiProviders), 1)
	assert.Equal(t, apiProviders[0].Name, "mokta")
}

func TestDeleteProvider(t *testing.T) {
	s := setupServer(t, withAdminIdentity)

	routes := s.GenerateRoutes(prometheus.NewRegistry())

	testProvider := &models.Provider{
		Name: "mokta",
	}

	err := data.CreateProvider(s.db, testProvider)
	assert.NilError(t, err)

	route := fmt.Sprintf("/v1/providers/%s", testProvider.ID)
	req, err := http.NewRequest(http.MethodDelete, route, nil)
	assert.NilError(t, err)

	req.Header.Add("Authorization", "Bearer "+adminAccessKey(s))

	resp := httptest.NewRecorder()
	routes.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusNoContent, resp.Code, resp.Body.String())
}

// withAdminIdentity may be used with setupServer to setup the server
// with an admin machine identity and access key
func withAdminIdentity(_ *testing.T, opts *Options) {
	opts.Identities = append(opts.Identities, Identity{
		Name:      "admin",
		AccessKey: "BlgpvURSGF.NdcemBdzxLTGIcjPXwPoZNrb",
	})
	opts.Grants = append(opts.Grants, Grant{
		Machine:  "admin",
		Role:     "admin",
		Resource: "infra",
	})
}

func TestDeleteProvider_NoDeleteInternalProvider(t *testing.T) {
	s := setupServer(t, withAdminIdentity)

	routes := s.GenerateRoutes(prometheus.NewRegistry())
	internalProvider := data.InfraProvider(s.db)

	route := fmt.Sprintf("/v1/providers/%s", internalProvider.ID)
	req, err := http.NewRequest(http.MethodDelete, route, nil)
	assert.NilError(t, err)

	req.Header.Add("Authorization", "Bearer "+adminAccessKey(s))

	resp := httptest.NewRecorder()
	routes.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusForbidden, resp.Code, resp.Body.String())
}

func TestCreateIdentity(t *testing.T) {
	s := setupServer(t)

	admin := &models.Identity{Name: "admin@example.com", Kind: models.UserKind}
	err := data.CreateIdentity(s.db, admin)
	assert.NilError(t, err)

	adminGrant := &models.Grant{
		Subject:   admin.PolyID(),
		Privilege: models.InfraAdminRole,
		Resource:  "infra",
	}
	err = data.CreateGrant(s.db, adminGrant)
	assert.NilError(t, err)

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Set("db", s.db)
	c.Set("identity", admin)

	handler := API{
		server: s,
	}

	t.Run("new unlinked user", func(t *testing.T) {
		req := &api.CreateIdentityRequest{
			Name: "test-create-identity@example.com",
			Kind: "user",
		}

		resp, err := handler.CreateIdentity(c, req)

		assert.NilError(t, err)
		assert.Equal(t, "test-create-identity@example.com", resp.Name)
		assert.Check(t, resp.OneTimePassword == "")
	})

	t.Run("new infra user gets one time password", func(t *testing.T) {
		req := &api.CreateIdentityRequest{
			Name:               "test-infra-identity@example.com",
			Kind:               "user",
			SetOneTimePassword: true,
		}

		resp, err := handler.CreateIdentity(c, req)
		assert.NilError(t, err)

		assert.NilError(t, err)
		assert.Equal(t, "test-infra-identity@example.com", resp.Name)
		assert.Check(t, resp.OneTimePassword != "")
	})

	t.Run("existing unlinked user gets password", func(t *testing.T) {
		req := &api.CreateIdentityRequest{
			Name: "test-link-identity@example.com",
			Kind: "user",
		}

		_, err := handler.CreateIdentity(c, req)
		assert.NilError(t, err)

		req = &api.CreateIdentityRequest{
			Name:               "test-link-identity@example.com",
			Kind:               "user",
			SetOneTimePassword: true,
		}

		resp, err := handler.CreateIdentity(c, req)
		assert.NilError(t, err)

		assert.NilError(t, err)
		assert.Equal(t, "test-link-identity@example.com", resp.Name)
		assert.Check(t, resp.OneTimePassword != "")
	})

	t.Run("new machine identities do not get one time password", func(t *testing.T) {
		req := &api.CreateIdentityRequest{
			Name:               "test-infra-machine-otp",
			Kind:               "machine",
			SetOneTimePassword: true,
		}

		resp, err := handler.CreateIdentity(c, req)
		assert.NilError(t, err)

		assert.NilError(t, err)
		assert.Equal(t, "test-infra-machine-otp", resp.Name)
		assert.Check(t, resp.OneTimePassword == "")
	})
}

func TestDeleteIdentity(t *testing.T) {
	s := setupServer(t, withAdminIdentity)

	routes := s.GenerateRoutes(prometheus.NewRegistry())

	testUser := &models.Identity{
		Name: "test",
		Kind: models.UserKind,
	}

	err := data.CreateIdentity(s.db, testUser)
	assert.NilError(t, err)

	route := fmt.Sprintf("/v1/identities/%s", testUser.ID)
	req, err := http.NewRequest(http.MethodDelete, route, nil)
	assert.NilError(t, err)

	req.Header.Add("Authorization", "Bearer "+adminAccessKey(s))

	resp := httptest.NewRecorder()
	routes.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusNoContent, resp.Code, resp.Body.String())
}

func TestDeleteIdentity_NoDeleteInternalIdentities(t *testing.T) {
	s := setupServer(t, withAdminIdentity)

	routes := s.GenerateRoutes(prometheus.NewRegistry())
	connector := data.InfraConnectorIdentity(s.db)

	route := fmt.Sprintf("/v1/identities/%s", connector.ID)
	req, err := http.NewRequest(http.MethodDelete, route, nil)
	assert.NilError(t, err)

	req.Header.Add("Authorization", "Bearer "+adminAccessKey(s))

	resp := httptest.NewRecorder()
	routes.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusForbidden, resp.Code, resp.Body.String())
}

func TestDeleteIdentity_NoDeleteSelf(t *testing.T) {
	s := setupServer(t)

	routes := s.GenerateRoutes(prometheus.NewRegistry())

	testUser := &models.Identity{
		Name: "test",
		Kind: models.UserKind,
	}

	err := data.CreateIdentity(s.db, testUser)
	assert.NilError(t, err)

	internalProvider, err := data.GetProvider(s.db, data.ByName(models.InternalInfraProviderName))
	assert.NilError(t, err)

	testAccessKey, err := data.CreateAccessKey(s.db, &models.AccessKey{
		Name:       "test",
		IssuedFor:  testUser.ID,
		ExpiresAt:  time.Now().Add(time.Hour),
		ProviderID: internalProvider.ID,
	})
	assert.NilError(t, err)

	route := fmt.Sprintf("/v1/identities/%s", testUser.ID)
	req, err := http.NewRequest(http.MethodDelete, route, nil)
	assert.NilError(t, err)

	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", testAccessKey))

	resp := httptest.NewRecorder()
	routes.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusForbidden, resp.Code, resp.Body.String())
}

func TestAPI_CreateGrant_Success(t *testing.T) {
	srv := setupServer(t, withAdminIdentity)
	routes := srv.GenerateRoutes(prometheus.NewRegistry())

	reqBody := strings.NewReader(`
		{
		  "subject": "i:12345",
		  "privilege": "admin-role",
		  "resource": "kubernetes.some-cluster"
		}`)

	resp := httptest.NewRecorder()
	req, err := http.NewRequest(http.MethodPost, "/v1/grants", reqBody)
	assert.NilError(t, err)
	req.Header.Add("Authorization", "Bearer "+adminAccessKey(srv))

	accessKey, err := data.ValidateAccessKey(srv.db, adminAccessKey(srv))
	assert.NilError(t, err)

	runStep(t, "response is ok", func(t *testing.T) {
		routes.ServeHTTP(resp, req)
		assert.Equal(t, resp.Code, http.StatusCreated)

		expected := jsonUnmarshal(t, fmt.Sprintf(`
		{
		  "id": "<any-valid-uid>",
		  "created_by": "%[1]v",
		  "privilege": "admin-role",
		  "resource": "kubernetes.some-cluster",
		  "subject": "i:12345",
		  "created": "%[2]v",
		  "updated": "%[2]v"
		}`,
			accessKey.IssuedFor,
			time.Now().UTC().Format(time.RFC3339),
		))
		actual := jsonUnmarshal(t, resp.Body.String())
		assert.DeepEqual(t, actual, expected, cmpAPIGrantJSON)
	})

	var newGrant api.Grant
	err = json.NewDecoder(resp.Body).Decode(&newGrant)
	assert.NilError(t, err)

	runStep(t, "grant exists", func(t *testing.T) {
		resp := httptest.NewRecorder()
		req, err := http.NewRequest(http.MethodGet, "/v1/grants/"+newGrant.ID.String(), nil)
		assert.NilError(t, err)
		req.Header.Add("Authorization", "Bearer "+adminAccessKey(srv))

		routes.ServeHTTP(resp, req)
		assert.Equal(t, resp.Code, http.StatusOK)

		var getGrant api.Grant
		err = json.NewDecoder(resp.Body).Decode(&getGrant)
		assert.NilError(t, err)
		assert.DeepEqual(t, getGrant, newGrant)
	})
}

var cmpAPIGrantJSON = gocmp.Options{
	gocmp.FilterPath(pathMapKey(`created`, `updated`), cmpApproximateTime),
	gocmp.FilterPath(pathMapKey(`id`), cmpAnyValidUID),
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
	return yd.Sub(xd) < 2*time.Second
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

func TestAPI_GetIdentity(t *testing.T) {
	srv := setupServer(t, withAdminIdentity)
	routes := srv.GenerateRoutes(prometheus.NewRegistry())

	createID := func(t *testing.T, name string, kind string) uid.ID {
		t.Helper()
		var buf bytes.Buffer
		body := api.CreateIdentityRequest{Name: name, Kind: kind}
		err := json.NewEncoder(&buf).Encode(body)
		assert.NilError(t, err)

		req, err := http.NewRequest(http.MethodPost, "/v1/identities", &buf)
		assert.NilError(t, err)
		req.Header.Add("Authorization", "Bearer "+adminAccessKey(srv))

		resp := httptest.NewRecorder()
		routes.ServeHTTP(resp, req)
		assert.Equal(t, resp.Code, http.StatusCreated, resp.Body.String())
		respObj := &api.CreateIdentityResponse{}
		err = json.Unmarshal(resp.Body.Bytes(), respObj)
		assert.NilError(t, err)
		return respObj.ID
	}
	id1 := createID(t, "me@example.com", "user")
	id2 := createID(t, "HAL", "machine")

	type testCase struct {
		urlPath  string
		setup    func(t *testing.T, req *http.Request)
		expected func(t *testing.T, resp *httptest.ResponseRecorder)
	}

	run := func(t *testing.T, tc testCase) {
		req, err := http.NewRequest(http.MethodGet, tc.urlPath, nil)
		assert.NilError(t, err)
		req.Header.Add("Authorization", "Bearer "+adminAccessKey(srv))

		if tc.setup != nil {
			tc.setup(t, req)
		}

		resp := httptest.NewRecorder()
		routes.ServeHTTP(resp, req)

		tc.expected(t, resp)
	}

	testCases := map[string]testCase{
		"not authenticated": {
			urlPath: "/v1/identities/" + id1.String(),
			setup: func(t *testing.T, req *http.Request) {
				req.Header.Del("Authorization")
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusUnauthorized)
			},
		},
		"not authorized": {
			urlPath: "/v1/identities/" + id2.String(),
			setup: func(t *testing.T, req *http.Request) {
				key, _ := createAccessKey(t, srv.db, "someonenew@example.com")

				req.Header.Set("Authorization", "Bearer "+key)
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusForbidden)
			},
		},
		"identity not found": {
			urlPath: "/v1/identities/1234",
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusNotFound)
			},
		},
		"identity by ID for self": {
			urlPath: "/v1/identities/" + id1.String(),
			setup: func(t *testing.T, req *http.Request) {
				token := &models.AccessKey{
					IssuedFor:  id1,
					ProviderID: data.InfraProvider(srv.db).ID,
					ExpiresAt:  time.Now().Add(10 * time.Second),
				}

				key, err := data.CreateAccessKey(srv.db, token)
				assert.NilError(t, err)

				req.Header.Set("Authorization", "Bearer "+key)
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusOK)
			},
		},
		"identity by ID for someone else": {
			urlPath: "/v1/identities/" + id1.String(),
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusOK)
			},
		},
		"identity by self": {
			urlPath: "/v1/identities/self",
			setup: func(t *testing.T, req *http.Request) {
				token := &models.AccessKey{
					IssuedFor:  id1,
					ProviderID: data.InfraProvider(srv.db).ID,
					ExpiresAt:  time.Now().Add(10 * time.Second),
				}

				key, err := data.CreateAccessKey(srv.db, token)
				assert.NilError(t, err)

				req.Header.Set("Authorization", "Bearer "+key)
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusOK, resp.Body)

				idResponse := api.Identity{}
				err := json.NewDecoder(resp.Body).Decode(&idResponse)
				assert.NilError(t, err)
				assert.Equal(t, idResponse.ID, id1)
			},
		},
		"full json response": {
			urlPath: "/v1/identities/" + id1.String(),
			setup: func(t *testing.T, req *http.Request) {
				req.Header.Set("Authorization", "Bearer "+adminAccessKey(srv))
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusOK)

				expected := jsonUnmarshal(t, fmt.Sprintf(`
					{
					  "id": "%[1]v",
					  "kind": "user",
					  "name": "me@example.com",
					  "lastSeenAt": "%[2]v",
					  "created": "%[2]v",
					  "updated": "%[2]v"
					}`,
					id1.String(),
					time.Now().UTC().Format(time.RFC3339),
				))
				actual := jsonUnmarshal(t, resp.Body.String())
				assert.DeepEqual(t, actual, expected, cmpAPIIdentityJSON)
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			run(t, tc)
		})
	}
}

var cmpAPIIdentityJSON = gocmp.Options{
	gocmp.FilterPath(pathMapKey(`created`, `updated`, `lastSeenAt`), cmpApproximateTime),
	gocmp.FilterPath(pathMapKey(`id`), cmpAnyValidUID),
}
