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

	createID := func(t *testing.T, name string) uid.ID {
		t.Helper()
		var buf bytes.Buffer
		body := api.CreateIdentityRequest{Name: name}
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
	id1 := createID(t, "me@example.com")
	id2 := createID(t, "other@example.com")
	id3 := createID(t, "HAL")
	_ = createID(t, "other-HAL")

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
				assert.Equal(t, resp.Body.String(), `{"items":[],"count":0}`)
			},
		},
		"name match": {
			urlPath: "/v1/identities?name=me@example.com",
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusOK)

				var actual api.ListResponse[api.Identity]
				err := json.NewDecoder(resp.Body).Decode(&actual)
				assert.NilError(t, err)
				expected := api.ListResponse[api.Identity]{
					Count: 1,
					Items: []api.Identity{
						{Name: "me@example.com"},
					},
				}
				assert.DeepEqual(t, actual, expected, cmpAPIIdentityShallow)
			},
		},
		"filter by ids": {
			urlPath: fmt.Sprintf("/v1/identities?ids=%s&ids=%s&ids=%s", id1, id2, id3),
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusOK)

				var actual api.ListResponse[api.Identity]
				err := json.NewDecoder(resp.Body).Decode(&actual)
				assert.NilError(t, err)
				expected := api.ListResponse[api.Identity]{
					Count: 3,
					Items: []api.Identity{
						{Name: "HAL"},
						{Name: "me@example.com"},
						{Name: "other@example.com"},
					},
				}
				assert.DeepEqual(t, actual, expected, cmpAPIIdentityShallow)
			},
		},
		"no filter": {
			urlPath: "/v1/identities",
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusOK)

				var actual api.ListResponse[api.Identity]
				err := json.NewDecoder(resp.Body).Decode(&actual)
				assert.NilError(t, err)
				expected := api.ListResponse[api.Identity]{
					Count: 6,
					Items: []api.Identity{
						{Name: "HAL"},
						{Name: "admin"},
						{Name: "connector"},
						{Name: "me@example.com"},
						{Name: "other-HAL"},
						{Name: "other@example.com"},
					},
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
	return x.Name == y.Name
})

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
		ExpiresAt:  time.Now().Add(5 * time.Minute),
	})
	assert.NilError(t, err)

	keys, err := handlers.ListAccessKeys(c, &api.ListAccessKeysRequest{})
	assert.NilError(t, err)

	assert.Assert(t, len(keys.Items) > 0)

	assert.Equal(t, keys.Items[0].IssuedForName, "foo@example.com")
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

	var apiProviders api.ListResponse[Provider]
	err = json.Unmarshal(resp.Body.Bytes(), &apiProviders)
	assert.NilError(t, err)

	assert.Equal(t, len(apiProviders.Items), 1)
	assert.Equal(t, apiProviders.Items[0].Name, "mokta")
}

func TestAPI_DeleteProvider(t *testing.T) {
	srv := setupServer(t, withAdminIdentity)
	routes := srv.GenerateRoutes(prometheus.NewRegistry())

	createProvider := func(t *testing.T) *models.Provider {
		t.Helper()
		p := &models.Provider{Name: "mokta"}
		err := data.CreateProvider(srv.db, p)
		assert.NilError(t, err)
		return p
	}

	provider1 := createProvider(t)

	type testCase struct {
		urlPath  string
		setup    func(t *testing.T, req *http.Request)
		expected func(t *testing.T, resp *httptest.ResponseRecorder)
	}

	run := func(t *testing.T, tc testCase) {
		req, err := http.NewRequest(http.MethodDelete, tc.urlPath, nil)
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
			urlPath: "/v1/providers/1234",
			setup: func(t *testing.T, req *http.Request) {
				req.Header.Del("Authorization")
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusUnauthorized, resp.Code, resp.Body.String())
			},
		},
		"not authorized": {
			urlPath: "/v1/providers/1234",
			setup: func(t *testing.T, req *http.Request) {
				key, _ := createAccessKey(t, srv.db, "someonenew@example.com")
				req.Header.Set("Authorization", "Bearer "+key)
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusForbidden, resp.Code, resp.Body.String())
			},
		},
		"successful delete": {
			urlPath: "/v1/providers/" + provider1.ID.String(),
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusNoContent, resp.Code, resp.Body.String())
			},
		},
		"infra provider can not be deleted": {
			urlPath: "/v1/providers/" + data.InfraProvider(srv.db).ID.String(),
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusForbidden, resp.Code, resp.Body.String())
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			run(t, tc)
		})
	}
}

// withAdminIdentity may be used with setupServer to setup the server
// with an admin identity and access key
func withAdminIdentity(_ *testing.T, opts *Options) {
	opts.Identities = append(opts.Identities, Identity{
		Name:      "admin",
		AccessKey: "BlgpvURSGF.NdcemBdzxLTGIcjPXwPoZNrb",
	})
	opts.Grants = append(opts.Grants, Grant{
		User:     "admin",
		Role:     "admin",
		Resource: "infra",
	})
}

func TestCreateIdentity(t *testing.T) {
	srv := setupServer(t, withAdminIdentity)
	routes := srv.GenerateRoutes(prometheus.NewRegistry())

	existing := &models.Identity{Name: "existing@example.com"}
	err := data.CreateIdentity(srv.db, existing)
	assert.NilError(t, err)

	type testCase struct {
		body     api.CreateIdentityRequest
		setup    func(t *testing.T, req *http.Request)
		expected func(t *testing.T, resp *httptest.ResponseRecorder)
	}

	run := func(t *testing.T, tc testCase) {
		body := jsonBody(t, tc.body)
		req, err := http.NewRequest(http.MethodPost, "/v1/identities", body)
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
			setup: func(t *testing.T, req *http.Request) {
				req.Header.Del("Authorization")
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusUnauthorized)
			},
		},
		"not authorized": {
			body: api.CreateIdentityRequest{
				Name: "noone@example.com",
			},
			setup: func(t *testing.T, req *http.Request) {
				key, _ := createAccessKey(t, srv.db, "someonenew@example.com")
				req.Header.Set("Authorization", "Bearer "+key)
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusForbidden, resp.Body.String())
			},
		},
		"missing required fields": {
			body: api.CreateIdentityRequest{},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusBadRequest, resp.Body.String())

				var apiError api.Error
				err := json.NewDecoder(resp.Body).Decode(&apiError)
				assert.NilError(t, err)
				assert.Equal(t, apiError.Message, "Name: is required")
			},
		},
		"create new unlinked user": {
			body: api.CreateIdentityRequest{Name: "test-create-identity@example.com"},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusCreated, resp.Code, resp.Body.String())

				var id api.CreateIdentityResponse
				err := json.NewDecoder(resp.Body).Decode(&id)
				assert.NilError(t, err)
				assert.Equal(t, "test-create-identity@example.com", id.Name)
				assert.Assert(t, id.OneTimePassword == "")
			},
		},
		"new infra user gets one time password": {
			body: api.CreateIdentityRequest{
				Name:               "test-infra-identity@example.com",
				SetOneTimePassword: true,
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusCreated, resp.Code, resp.Body.String())

				var id api.CreateIdentityResponse
				err := json.NewDecoder(resp.Body).Decode(&id)
				assert.NilError(t, err)
				assert.Equal(t, "test-infra-identity@example.com", id.Name)
				assert.Assert(t, id.OneTimePassword != "")
			},
		},
		"existing unlinked user gets password": {
			body: api.CreateIdentityRequest{
				Name:               "existing@example.com",
				SetOneTimePassword: true,
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusCreated, resp.Code, resp.Body.String())

				var id api.CreateIdentityResponse
				err := json.NewDecoder(resp.Body).Decode(&id)
				assert.NilError(t, err)
				assert.Equal(t, "existing@example.com", id.Name)
				assert.Assert(t, id.OneTimePassword != "")
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			run(t, tc)
		})
	}
}

func jsonBody(t *testing.T, body interface{}) *bytes.Buffer {
	t.Helper()
	buf := new(bytes.Buffer)
	err := json.NewEncoder(buf).Encode(body)
	assert.NilError(t, err)
	return buf
}

func TestDeleteIdentity(t *testing.T) {
	s := setupServer(t, withAdminIdentity)

	routes := s.GenerateRoutes(prometheus.NewRegistry())

	testUser := &models.Identity{Name: "test"}

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

	testUser := &models.Identity{Name: "test"}

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

	createID := func(t *testing.T, name string) uid.ID {
		t.Helper()
		var buf bytes.Buffer
		body := api.CreateIdentityRequest{Name: name}
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
	idMe := createID(t, "me@example.com")
	idHal := createID(t, "HAL")

	token := &models.AccessKey{
		IssuedFor:  idMe,
		ProviderID: data.InfraProvider(srv.db).ID,
		ExpiresAt:  time.Now().Add(10 * time.Second),
	}

	accessKeyMe, err := data.CreateAccessKey(srv.db, token)
	assert.NilError(t, err)

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
			urlPath: "/v1/identities/" + idMe.String(),
			setup: func(t *testing.T, req *http.Request) {
				req.Header.Del("Authorization")
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusUnauthorized)
			},
		},
		"not authorized": {
			urlPath: "/v1/identities/" + idHal.String(),
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
			urlPath: "/v1/identities/" + idMe.String(),
			setup: func(t *testing.T, req *http.Request) {
				req.Header.Set("Authorization", "Bearer "+accessKeyMe)
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusOK)
			},
		},
		"identity by ID for someone else": {
			urlPath: "/v1/identities/" + idMe.String(),
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusOK)
			},
		},
		"identity by self": {
			urlPath: "/v1/identities/self",
			setup: func(t *testing.T, req *http.Request) {
				token := &models.AccessKey{
					IssuedFor:  idMe,
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
				assert.Equal(t, idResponse.ID, idMe)
			},
		},
		"full json response": {
			urlPath: "/v1/identities/" + idMe.String(),
			setup: func(t *testing.T, req *http.Request) {
				req.Header.Set("Authorization", "Bearer "+accessKeyMe)
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusOK)

				expected := jsonUnmarshal(t, fmt.Sprintf(`
					{
					  "id": "%[1]v",
					  "name": "me@example.com",
					  "lastSeenAt": "%[2]v",
					  "created": "%[2]v",
					  "updated": "%[2]v"
					}`,
					idMe.String(),
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

func TestAPI_CreateAccessKey(t *testing.T) {
	srv := setupServer(t, withAdminIdentity)
	routes := srv.GenerateRoutes(prometheus.NewRegistry())

	connector := data.InfraConnectorIdentity(srv.db)

	run := func(body api.CreateAccessKeyRequest) *api.CreateAccessKeyResponse {
		var buf bytes.Buffer
		err := json.NewEncoder(&buf).Encode(body)
		assert.NilError(t, err)

		req, err := http.NewRequest(http.MethodPost, "/v1/access-keys", &buf)
		assert.NilError(t, err)
		req.Header.Add("Authorization", "Bearer "+adminAccessKey(srv))

		resp := httptest.NewRecorder()
		routes.ServeHTTP(resp, req)

		assert.Equal(t, resp.Code, http.StatusCreated, resp.Body.String())

		respBody := &api.CreateAccessKeyResponse{}
		err = json.Unmarshal(resp.Body.Bytes(), respBody)
		assert.NilError(t, err)

		return respBody
	}

	t.Run("AutomaticName", func(t *testing.T) {
		req := api.CreateAccessKeyRequest{
			IdentityID:        connector.ID,
			TTL:               api.Duration(time.Minute),
			ExtensionDeadline: api.Duration(time.Minute),
		}

		resp := run(req)
		assert.Assert(t, strings.HasPrefix(resp.Name, "connector-"))
	})

	t.Run("UserProvidedName", func(t *testing.T) {
		req := api.CreateAccessKeyRequest{
			IdentityID:        connector.ID,
			Name:              "mysupersecretaccesskey",
			TTL:               api.Duration(time.Minute),
			ExtensionDeadline: api.Duration(time.Minute),
		}

		resp := run(req)
		assert.Equal(t, resp.Name, "mysupersecretaccesskey")
	})
}

func TestAPI_DeleteGrant(t *testing.T) {
	srv := setupServer(t, withAdminIdentity)
	routes := srv.GenerateRoutes(prometheus.NewRegistry())

	user := &models.Identity{Name: "non-admin"}

	err := data.CreateIdentity(srv.db, user)
	assert.NilError(t, err)

	t.Run("last infra admin is deleted", func(t *testing.T) {
		infraAdminGrants, err := data.ListGrants(srv.db, data.ByPrivilege(models.InfraAdminRole), data.ByResource("infra"))
		assert.NilError(t, err)
		assert.Assert(t, len(infraAdminGrants) == 1)

		req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/v1/grants/%s", infraAdminGrants[0].ID), nil)
		req.Header.Set("Authorization", "Bearer "+adminAccessKey(srv))

		resp := httptest.NewRecorder()
		routes.ServeHTTP(resp, req)

		assert.Equal(t, resp.Code, http.StatusForbidden, resp.Body.String())
	})

	t.Run("not last infra admin is deleted", func(t *testing.T) {
		grant2 := &models.Grant{
			Subject:   uid.NewIdentityPolymorphicID(user.ID),
			Privilege: models.InfraAdminRole,
			Resource:  "infra",
		}

		err := data.CreateGrant(srv.db, grant2)
		assert.NilError(t, err)

		req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/v1/grants/%s", grant2.ID), nil)
		req.Header.Set("Authorization", "Bearer "+adminAccessKey(srv))

		resp := httptest.NewRecorder()
		routes.ServeHTTP(resp, req)

		assert.Equal(t, resp.Code, http.StatusNoContent, resp.Body.String())
	})

	t.Run("last infra non-admin is deleted", func(t *testing.T) {
		grant2 := &models.Grant{
			Subject:   uid.NewIdentityPolymorphicID(user.ID),
			Privilege: models.InfraViewRole,
			Resource:  "infra",
		}

		err := data.CreateGrant(srv.db, grant2)
		assert.NilError(t, err)

		req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/v1/grants/%s", grant2.ID), nil)
		req.Header.Set("Authorization", "Bearer "+adminAccessKey(srv))

		resp := httptest.NewRecorder()
		routes.ServeHTTP(resp, req)

		assert.Equal(t, resp.Code, http.StatusNoContent, resp.Body.String())
	})

	t.Run("last non-infra admin is deleted", func(t *testing.T) {
		grant2 := &models.Grant{
			Subject:   uid.NewIdentityPolymorphicID(user.ID),
			Privilege: "admin",
			Resource:  "kubernetes.example",
		}

		err := data.CreateGrant(srv.db, grant2)
		assert.NilError(t, err)

		req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/v1/grants/%s", grant2.ID), nil)
		req.Header.Set("Authorization", "Bearer "+adminAccessKey(srv))

		resp := httptest.NewRecorder()
		routes.ServeHTTP(resp, req)

		assert.Equal(t, resp.Code, http.StatusNoContent, resp.Body.String())
	})
}
