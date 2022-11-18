package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"
	"time"

	gocmp "github.com/google/go-cmp/cmp"
	"gotest.tools/v3/assert"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

func TestAPI_CreateAccessKey(t *testing.T) {
	type testCase struct {
		name     string
		setup    func(t *testing.T) api.CreateAccessKeyRequest
		expected func(t *testing.T, response *httptest.ResponseRecorder)
	}

	srv := setupServer(t, withAdminUser)
	routes := srv.GenerateRoutes()

	userResp := createUser(t, srv, routes, "usera@example.com")

	run := func(t *testing.T, tc testCase) {
		body := tc.setup(t)

		// nolint:noctx
		req := httptest.NewRequest(http.MethodPost, "/api/access-keys", jsonBody(t, body))
		req.Header.Set("Authorization", "Bearer "+adminAccessKey(srv))
		req.Header.Set("Infra-Version", apiVersionLatest)

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
					{FieldName: "name", Errors: []string{"character '/' at position 34 is not allowed"}},
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

func TestAPI_ListAccessKeys_Success(t *testing.T) {
	srv := setupServer(t, withAdminUser)
	routes := srv.GenerateRoutes()

	run := func() api.ListResponse[api.AccessKey] {
		req := httptest.NewRequest(http.MethodGet, "/api/access-keys", nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", adminAccessKey(srv)))
		req.Header.Set("Infra-Version", apiVersionLatest)

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
}

func TestAPI_ListAccessKeys(t *testing.T) {
	srv := setupServer(t, withAdminUser)
	routes := srv.GenerateRoutes()

	db := srv.DB()

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

	ak1 := &models.AccessKey{
		Name:       "foo",
		IssuedFor:  user.ID,
		ProviderID: provider.ID,
		ExpiresAt:  time.Now().UTC().Add(5 * time.Minute),
	}
	_, err = data.CreateAccessKey(db, ak1)
	assert.NilError(t, err)

	ak2 := &models.AccessKey{
		Name:       "expired",
		IssuedFor:  user.ID,
		ProviderID: provider.ID,
		ExpiresAt:  time.Now().UTC().Add(-5 * time.Minute),
	}
	_, err = data.CreateAccessKey(db, ak2)
	assert.NilError(t, err)

	ak3 := &models.AccessKey{
		Name:              "not_extended",
		IssuedFor:         user.ID,
		ProviderID:        provider.ID,
		ExpiresAt:         time.Now().UTC().Add(5 * time.Minute),
		ExtensionDeadline: time.Now().UTC().Add(-5 * time.Minute),
	}
	_, err = data.CreateAccessKey(db, ak3)
	assert.NilError(t, err)

	cmpAccessKeyShallow := gocmp.Comparer(func(x, y api.AccessKey) bool {
		return x.ID == y.ID && x.IssuedForName == y.IssuedForName
	})

	t.Run("success", func(t *testing.T) {
		resp := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/access-keys", nil)
		req.Header.Set("Authorization", "Bearer "+adminAccessKey(srv))
		req.Header.Set("Infra-Version", apiVersionLatest)

		routes.ServeHTTP(resp, req)
		assert.Equal(t, resp.Code, http.StatusOK)

		keys := api.ListResponse[api.AccessKey]{}
		err = json.Unmarshal(resp.Body.Bytes(), &keys)
		assert.NilError(t, err)

		// TODO: replace this with a more strict assertion using DeepEqual
		assert.Equal(t, len(keys.Items), 2)
		for _, item := range keys.Items {
			assert.Assert(t, item.Expires.Time().UTC().After(time.Now().UTC()) || item.Expires.Time().IsZero())
			assert.Assert(t, item.ExtensionDeadline.Time().UTC().After(time.Now().UTC()) || item.ExtensionDeadline.Time().IsZero())
		}
	})

	t.Run("success w/ pagination", func(t *testing.T) {
		resp := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/access-keys?limit=2&user_id="+user.ID.String(), nil)
		req.Header.Set("Authorization", "Bearer "+adminAccessKey(srv))
		req.Header.Set("Infra-Version", "0.13.0") // use older version of the API

		routes.ServeHTTP(resp, req)
		assert.Equal(t, resp.Code, http.StatusOK)

		keys := api.ListResponse[api.AccessKey]{}
		err = json.Unmarshal(resp.Body.Bytes(), &keys)
		assert.NilError(t, err)

		expected := api.ListResponse[api.AccessKey]{
			Count:              1,
			PaginationResponse: api.PaginationResponse{Limit: 2, Page: 1, TotalCount: 1, TotalPages: 1},
			Items: []api.AccessKey{
				{ID: ak1.ID, IssuedForName: user.Name},
			},
		}

		assert.Equal(t, keys.PaginationResponse, api.PaginationResponse{Limit: 2, Page: 1, TotalCount: 1, TotalPages: 1})
		assert.DeepEqual(t, keys.Items, expected.Items, cmpAccessKeyShallow)

		for _, item := range keys.Items {
			assert.Assert(t, item.Expires.Time().UTC().After(time.Now().UTC()) || item.Expires.Time().IsZero())
			assert.Assert(t, item.ExtensionDeadline.Time().UTC().After(time.Now().UTC()) || item.ExtensionDeadline.Time().IsZero())
		}
	})

	t.Run("delete by name", func(t *testing.T) {
		key := &models.AccessKey{
			Name:       "deleteme",
			IssuedFor:  user.ID,
			ProviderID: provider.ID,
			ExpiresAt:  time.Now().Add(5 * time.Minute),
		}
		_, err := data.CreateAccessKey(srv.db, key)
		assert.NilError(t, err)

		resp := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodDelete, "/api/access-keys?name=deleteme", nil)
		req.Header.Set("Authorization", "Bearer "+ak1.Token())
		req.Header.Set("Infra-Version", apiVersionLatest)

		routes.ServeHTTP(resp, req)
		assert.Equal(t, resp.Code, http.StatusNoContent)
	})

	t.Run("do not allow delete of the key used in the request", func(t *testing.T) {
		resp := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodDelete, "/api/access-keys/"+ak1.ID.String(), nil)
		req.Header.Set("Authorization", "Bearer "+ak1.Token())
		req.Header.Set("Infra-Version", apiVersionLatest)

		routes.ServeHTTP(resp, req)
		assert.Equal(t, resp.Code, http.StatusBadRequest)
	})

	t.Run("delete by name as non-admin", func(t *testing.T) {
		user := &models.Identity{
			Model:              models.Model{ID: uid.New()},
			Name:               "deletemyownkey@example.com",
			OrganizationMember: models.OrganizationMember{OrganizationID: provider.OrganizationID},
		}
		err := data.CreateIdentity(db, user)
		assert.NilError(t, err)

		key1 := &models.AccessKey{
			Name:               "deletemetoo",
			IssuedFor:          user.ID,
			ProviderID:         provider.ID,
			ExpiresAt:          time.Now().Add(5 * time.Minute),
			OrganizationMember: models.OrganizationMember{OrganizationID: provider.OrganizationID},
		}
		_, err = data.CreateAccessKey(srv.db, key1)
		assert.NilError(t, err)

		key2 := &models.AccessKey{
			Name:               "deletemetoo2",
			IssuedFor:          user.ID,
			ProviderID:         provider.ID,
			ExpiresAt:          time.Now().Add(5 * time.Minute),
			OrganizationMember: models.OrganizationMember{OrganizationID: provider.OrganizationID},
		}
		_, err = data.CreateAccessKey(srv.db, key2)
		assert.NilError(t, err)

		resp := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodDelete, "/api/access-keys?name=deletemetoo2", nil)
		req.Header.Set("Authorization", "Bearer "+key1.Token())
		req.Header.Set("Infra-Version", apiVersionLatest)

		routes.ServeHTTP(resp, req)
		assert.Equal(t, resp.Code, http.StatusNoContent)
	})

	t.Run("delete by name missing", func(t *testing.T) {
		resp := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodDelete, "/api/access-keys?name=deletesomething", nil)
		req.Header.Set("Authorization", "Bearer "+ak1.Token())
		req.Header.Set("Infra-Version", apiVersionLatest)

		routes.ServeHTTP(resp, req)
		assert.Equal(t, resp.Code, http.StatusNotFound)
	})

	t.Run("show expired", func(t *testing.T) {
		resp := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/access-keys?showExpired=1", nil)
		req.Header.Set("Authorization", "Bearer "+adminAccessKey(srv))
		req.Header.Set("Infra-Version", apiVersionLatest)

		routes.ServeHTTP(resp, req)
		assert.Equal(t, resp.Code, http.StatusOK)

		keys := api.ListResponse[api.AccessKey]{}
		err := json.Unmarshal(resp.Body.Bytes(), &keys)
		assert.NilError(t, err)

		// TODO: replace this with a more strict assertion using DeepEqual
		assert.Equal(t, len(keys.Items), 5)

		sort.SliceIsSorted(keys.Items, func(i, j int) bool {
			return keys.Items[i].Name < keys.Items[j].Name
		})
	})

	t.Run("latest", func(t *testing.T) {
		resp := httptest.NewRecorder()
		// nolint:noctx
		req := httptest.NewRequest(http.MethodGet, "/api/access-keys", nil)
		req.Header.Set("Authorization", "Bearer "+adminAccessKey(srv))
		req.Header.Set("Infra-Version", "0.12.3")

		routes.ServeHTTP(resp, req)
		assert.Equal(t, resp.Code, http.StatusOK)

		resp1 := &api.ListResponse[api.AccessKey]{}
		err := json.Unmarshal(resp.Body.Bytes(), resp1)
		assert.NilError(t, err)

		assert.Assert(t, len(resp1.Items) > 0)
	})

	t.Run("no version header", func(t *testing.T) {
		resp := httptest.NewRecorder()
		// nolint:noctx
		req := httptest.NewRequest(http.MethodGet, "/api/access-keys", nil)
		req.Header.Set("Authorization", "Bearer "+adminAccessKey(srv))

		routes.ServeHTTP(resp, req)
		assert.Equal(t, resp.Code, http.StatusBadRequest)

		errMsg := api.Error{}
		err := json.Unmarshal(resp.Body.Bytes(), &errMsg)
		assert.NilError(t, err)

		assert.Assert(t, strings.Contains(errMsg.Message, "Infra-Version header is required"))
		assert.Equal(t, errMsg.Code, int32(400))
	})
}
