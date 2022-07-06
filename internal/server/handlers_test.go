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
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"gotest.tools/v3/assert"
	"k8s.io/utils/strings/slices"

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

var defaultPagination api.PaginationResponse

func TestAPI_ListUsers(t *testing.T) {
	srv := setupServer(t, withAdminUser)
	routes := srv.GenerateRoutes(prometheus.NewRegistry())

	// TODO: Convert the "humans" group and "AnotherUser" user to call the standard http endpoints
	//       when the new endpoint to add a user to a group exists
	humans := models.Group{Name: "humans"}
	createGroups(t, srv.db, &humans)
	anotherID := models.Identity{
		Name:   "AnotherUser@example.com",
		Groups: []models.Group{humans},
	}
	createIdentities(t, srv.db, &anotherID)

	createID := func(t *testing.T, name string) uid.ID {
		t.Helper()
		var buf bytes.Buffer
		body := api.CreateUserRequest{Name: name}
		err := json.NewEncoder(&buf).Encode(body)
		assert.NilError(t, err)

		req, err := http.NewRequest(http.MethodPost, "/api/users", &buf)
		assert.NilError(t, err)
		req.Header.Add("Authorization", "Bearer "+adminAccessKey(srv))
		req.Header.Add("Infra-Version", "0.12.3")

		resp := httptest.NewRecorder()
		routes.ServeHTTP(resp, req)
		assert.Equal(t, resp.Code, http.StatusCreated, resp.Body.String())
		respObj := &api.CreateUserResponse{}
		err = json.Unmarshal(resp.Body.Bytes(), respObj)
		assert.NilError(t, err)
		return respObj.ID
	}
	id1 := createID(t, "me@example.com")
	id2 := createID(t, "other@example.com")
	id3 := createID(t, "HAL@example.com")
	_ = createID(t, "other-HAL@example.com")

	type testCase struct {
		urlPath  string
		setup    func(t *testing.T, req *http.Request)
		expected func(t *testing.T, resp *httptest.ResponseRecorder)
	}

	run := func(t *testing.T, tc testCase) {
		req, err := http.NewRequest(http.MethodGet, tc.urlPath, nil)
		assert.NilError(t, err)
		req.Header.Add("Authorization", "Bearer "+adminAccessKey(srv))
		req.Header.Add("Infra-Version", "0.12.3")

		if tc.setup != nil {
			tc.setup(t, req)
		}

		resp := httptest.NewRecorder()
		routes.ServeHTTP(resp, req)

		tc.expected(t, resp)
	}

	testCases := map[string]testCase{
		"no name match": {
			urlPath: "/api/users?name=doesnotmatch",
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusOK)
				assert.Equal(t, resp.Body.String(), `{"count":0,"items":[]}`)
			},
		},
		"name match": {
			urlPath: "/api/users?name=me@example.com",
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusOK)

				var actual api.ListResponse[api.User]
				err := json.NewDecoder(resp.Body).Decode(&actual)
				assert.NilError(t, err)
				expected := api.ListResponse[api.User]{
					Count: 1,
					Items: []api.User{
						{Name: "me@example.com"},
					},
					PaginationResponse: defaultPagination,
				}
				assert.DeepEqual(t, actual, expected, cmpAPIUserShallow)
			},
		},
		"filter by ids": {
			urlPath: fmt.Sprintf("/api/users?ids=%s&ids=%s&ids=%s", id1, id2, id3),
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusOK)

				var actual api.ListResponse[api.User]
				err := json.NewDecoder(resp.Body).Decode(&actual)
				assert.NilError(t, err)
				expected := api.ListResponse[api.User]{
					Count: 3,
					Items: []api.User{
						{Name: "HAL@example.com"},
						{Name: "me@example.com"},
						{Name: "other@example.com"},
					},
					PaginationResponse: defaultPagination,
				}
				assert.DeepEqual(t, actual, expected, cmpAPIUserShallow)
			},
		},
		"no filter": {
			urlPath: "/api/users",
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusOK)

				var actual api.ListResponse[api.User]
				err := json.NewDecoder(resp.Body).Decode(&actual)
				assert.NilError(t, err)
				expected := api.ListResponse[api.User]{
					Count: 7,
					Items: []api.User{
						{Name: "AnotherUser@example.com"},
						{Name: "HAL@example.com"},
						{Name: "admin@example.com"},
						{Name: "connector"},
						{Name: "me@example.com"},
						{Name: "other-HAL@example.com"},
						{Name: "other@example.com"},
					},
					PaginationResponse: defaultPagination,
				}
				assert.DeepEqual(t, actual, expected, cmpAPIUserShallow)
			},
		},
		"no authorization": {
			urlPath: "/api/users",
			setup: func(t *testing.T, req *http.Request) {
				req.Header.Del("Authorization")
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusUnauthorized)
			},
		},
		"page 2 limit 2": {
			urlPath: "/api/users?limit=2&page=2",
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusOK)

				var actual api.ListResponse[api.User]
				err := json.NewDecoder(resp.Body).Decode(&actual)
				assert.NilError(t, err)
				expected := api.ListResponse[api.User]{
					Count: 2,
					Items: []api.User{
						{Name: "admin@example.com"},
						{Name: "connector"},
					},
					PaginationResponse: api.PaginationResponse{
						Page:       2,
						Limit:      2,
						TotalPages: 4,
						TotalCount: 7,
					},
				}
				assert.DeepEqual(t, actual, expected, cmpAPIUserShallow)
			},
		},
		"user in group": {
			urlPath: fmt.Sprintf("/api/users?group=%s", humans.ID),
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusOK)

				var actual api.ListResponse[api.User]
				err := json.NewDecoder(resp.Body).Decode(&actual)
				assert.NilError(t, err)

				expected := api.ListResponse[api.User]{
					Count: 1,
					Items: []api.User{
						{Name: anotherID.Name},
					},
					PaginationResponse: defaultPagination,
				}
				assert.DeepEqual(t, actual, expected, cmpAPIUserShallow)
			},
		},
		"invalid limit": {
			urlPath: "/api/users?limit=1001",
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusBadRequest)
			},
		},
		"invalid page": {
			urlPath: "/api/users?page=-1",
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusBadRequest)
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

var cmpAPIUserShallow = gocmp.Comparer(func(x, y api.User) bool {
	return x.Name == y.Name
})

func TestAPI_GetUserProviderNameResponse(t *testing.T) {
	srv := setupServer(t, withAdminUser)
	routes := srv.GenerateRoutes(prometheus.NewRegistry())

	user := &models.Identity{Name: "steve"}
	err := data.CreateIdentity(srv.db, user)
	assert.NilError(t, err)

	p := data.InfraProvider(srv.db)

	_, err = data.CreateProviderUser(srv.db, p, user)
	assert.NilError(t, err)

	req, err := http.NewRequest(http.MethodGet, "/api/users/"+user.ID.String(), nil)
	assert.NilError(t, err)
	req.Header.Add("Authorization", "Bearer "+adminAccessKey(srv))
	req.Header.Add("Infra-Version", "0.13.3")

	resp := httptest.NewRecorder()
	routes.ServeHTTP(resp, req)

	t.Log(resp.Body.String())
	assert.Equal(t, 200, resp.Code)

	u := &api.User{}

	err = json.Unmarshal(resp.Body.Bytes(), u)
	assert.NilError(t, err)
	assert.DeepEqual(t, []string{"infra"}, u.ProviderNames)
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
		assert.Equal(t, resp.Code, http.StatusOK)

		resp1 := []api.AccessKey{}
		err = json.Unmarshal(resp.Body.Bytes(), &resp1)
		assert.NilError(t, err)

		assert.Assert(t, len(resp1) > 0)
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

func TestListProviders(t *testing.T) {
	s := setupServer(t, withAdminUser)
	routes := s.GenerateRoutes(prometheus.NewRegistry())

	testProvider := &models.Provider{Name: "mokta", Kind: models.ProviderKindOkta, AuthURL: "https://example.com/v1/auth", Scopes: []string{"openid", "email"}}

	err := data.CreateProvider(s.db, testProvider)
	assert.NilError(t, err)

	dbProviders, err := data.ListProviders(s.db, &models.Pagination{})
	assert.NilError(t, err)
	assert.Equal(t, len(dbProviders), 2)

	req, err := http.NewRequest(http.MethodGet, "/api/providers", nil)
	assert.NilError(t, err)

	req.Header.Add("Authorization", "Bearer "+adminAccessKey(s))
	req.Header.Add("Infra-Version", "0.12.3")

	resp := httptest.NewRecorder()
	routes.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code, resp.Body.String())

	var apiProviders api.ListResponse[Provider]
	err = json.Unmarshal(resp.Body.Bytes(), &apiProviders)
	assert.NilError(t, err)

	assert.Equal(t, len(apiProviders.Items), 1)
	assert.Equal(t, apiProviders.Items[0].Name, "mokta")
	assert.Equal(t, apiProviders.Items[0].AuthURL, "https://example.com/v1/auth")
	assert.Assert(t, slices.Equal(apiProviders.Items[0].Scopes, []string{"openid", "email"}))
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

func TestCreateIdentity(t *testing.T) {
	srv := setupServer(t, withAdminUser)
	routes := srv.GenerateRoutes(prometheus.NewRegistry())

	existing := &models.Identity{Name: "existing@example.com"}
	err := data.CreateIdentity(srv.db, existing)
	assert.NilError(t, err)

	type testCase struct {
		body     api.CreateUserRequest
		setup    func(t *testing.T, req *http.Request)
		expected func(t *testing.T, resp *httptest.ResponseRecorder)
	}

	run := func(t *testing.T, tc testCase) {
		body := jsonBody(t, tc.body)
		req, err := http.NewRequest(http.MethodPost, "/api/users", body)
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
			body: api.CreateUserRequest{
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
			body: api.CreateUserRequest{},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusBadRequest, resp.Body.String())

				var apiError api.Error
				err := json.NewDecoder(resp.Body).Decode(&apiError)
				assert.NilError(t, err)
				assert.Equal(t, apiError.Message, "Name: failed the \"email\" check")
			},
		},
		"create new unlinked user": {
			body: api.CreateUserRequest{Name: "test-create-identity@example.com"},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusCreated, resp.Code, resp.Body.String())

				var id api.CreateUserResponse
				err := json.NewDecoder(resp.Body).Decode(&id)
				assert.NilError(t, err)
				assert.Equal(t, "test-create-identity@example.com", id.Name)
			},
		},
		"new infra user gets one time password": {
			body: api.CreateUserRequest{
				Name: "test-infra-identity@example.com",
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusCreated, resp.Code, resp.Body.String())

				var id api.CreateUserResponse
				err := json.NewDecoder(resp.Body).Decode(&id)
				assert.NilError(t, err)
				assert.Equal(t, "test-infra-identity@example.com", id.Name)
				assert.Assert(t, id.OneTimePassword != "")
			},
		},
		"existing unlinked user gets password": {
			body: api.CreateUserRequest{
				Name: "existing@example.com",
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusCreated, resp.Code, resp.Body.String())

				var id api.CreateUserResponse
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

// Note this test is the result of a long conversation, don't change lightly.
func TestCreateUserAndUpdatePassword(t *testing.T) {
	db := setupDB(t)
	a := &API{server: &Server{db: db}}
	admin := createAdmin(t, db)

	t.Run("with an IDP user existing", func(t *testing.T) {
		idp := &models.Provider{Name: "Super Provider", Kind: models.ProviderKindOIDC}
		err := data.CreateProvider(db, idp)
		assert.NilError(t, err)

		user := &models.Identity{Name: "user@example.com"}

		err = data.CreateIdentity(db, user)
		assert.NilError(t, err)

		_, err = data.CreateProviderUser(db, idp, user)
		assert.NilError(t, err)

		t.Run("as an admin", func(t *testing.T) {
			ctx := loginAs(db, admin)
			t.Run("I can set passwords for IDP users ", func(t *testing.T) {
				// (which creates the infra user)
				_, err := a.UpdateUser(ctx, &api.UpdateUserRequest{
					ID:       user.ID,
					Password: "1234567890987654321a!",
				})
				assert.NilError(t, err)
				_, err = data.GetProviderUser(db, data.InfraProvider(db).ID, user.ID)
				assert.NilError(t, err)
				cred, err := data.GetCredential(db, data.ByIdentityID(user.ID))
				assert.NilError(t, err)
				assert.Equal(t, true, cred.OneTimePassword)
			})
		})
		t.Run("as a user", func(t *testing.T) {
			ctx := loginAs(db, user)
			t.Run("with no existing infra user", func(t *testing.T) {
				err = data.DeleteProviderUsers(db, data.ByIdentityID(user.ID), data.ByProviderID(data.InfraProvider(db).ID))
				assert.NilError(t, err)

				cred, _ := data.GetCredential(db, data.ByIdentityID(user.ID))
				if cred != nil {
					_ = data.DeleteCredential(db, cred.ID)
				}

				t.Run("I cannot set a password", func(t *testing.T) {
					_, err := a.UpdateUser(ctx, &api.UpdateUserRequest{
						ID:       user.ID,
						Password: "1234567890987654321a!",
					})
					assert.Error(t, err, "existing credential: record not found")
				})
			})
			t.Run("with an existing infra user", func(t *testing.T) {
				_, _ = data.CreateProviderUser(db, data.InfraProvider(db), user)

				_ = data.CreateCredential(db, &models.Credential{
					IdentityID:   user.ID,
					PasswordHash: []byte("random password"),
				})

				t.Run("I can change my password", func(t *testing.T) {
					_, err := a.UpdateUser(ctx, &api.UpdateUserRequest{
						ID:       user.ID,
						Password: "1234567890987654321a!",
					})
					assert.NilError(t, err)
				})
			})
		})
	})
	t.Run("without an IDP user existing", func(t *testing.T) {
		t.Run("as an admin", func(t *testing.T) {
			ctx := loginAs(db, admin)
			var tmpUserID uid.ID

			t.Run("I can create a user", func(t *testing.T) {
				resp, err := a.CreateUser(ctx, &api.CreateUserRequest{
					Name: "joe+" + generate.MathRandom(10, generate.CharsetAlphaNumeric),
				})
				tmpUserID = resp.ID
				assert.NilError(t, err)
			})

			t.Run("I can change a password for a user", func(t *testing.T) {
				_, err := a.UpdateUser(ctx, &api.UpdateUserRequest{
					ID:       tmpUserID,
					Password: "123454676twefdhsds",
				})
				assert.NilError(t, err)
			})
		})
		t.Run("as a user", func(t *testing.T) {
			user := &models.Identity{Name: "user2@example.com"}

			err := data.CreateIdentity(db, user)
			assert.NilError(t, err)

			_, err = data.CreateProviderUser(db, data.InfraProvider(db), user)
			assert.NilError(t, err)

			err = data.CreateCredential(db, &models.Credential{
				IdentityID:   user.ID,
				PasswordHash: []byte("random password"),
			})
			assert.NilError(t, err)

			ctx := loginAs(db, user)
			t.Run("I can change my password", func(t *testing.T) {
				_, err := a.UpdateUser(ctx, &api.UpdateUserRequest{
					ID:       user.ID,
					Password: "123454676twefdhsds",
				})
				assert.NilError(t, err)
			})
		})
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

func TestDeleteUser(t *testing.T) {
	s := setupServer(t, withAdminUser)

	routes := s.GenerateRoutes(prometheus.NewRegistry())

	testUser := &models.Identity{Name: "test"}

	err := data.CreateIdentity(s.db, testUser)
	assert.NilError(t, err)

	route := fmt.Sprintf("/api/users/%s", testUser.ID)
	req, err := http.NewRequest(http.MethodDelete, route, nil)
	assert.NilError(t, err)

	req.Header.Add("Authorization", "Bearer "+adminAccessKey(s))

	resp := httptest.NewRecorder()
	routes.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusNoContent, resp.Code, resp.Body.String())
}

func TestDeleteUser_NoDeleteInternalIdentities(t *testing.T) {
	s := setupServer(t, withAdminUser)

	routes := s.GenerateRoutes(prometheus.NewRegistry())
	connector := data.InfraConnectorIdentity(s.db)

	route := fmt.Sprintf("/api/users/%s", connector.ID)
	req, err := http.NewRequest(http.MethodDelete, route, nil)
	assert.NilError(t, err)

	req.Header.Add("Authorization", "Bearer "+adminAccessKey(s))

	resp := httptest.NewRecorder()
	routes.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusBadRequest, resp.Code, resp.Body.String())
}

func TestDeleteUser_NoDeleteSelf(t *testing.T) {
	s := setupServer(t)

	routes := s.GenerateRoutes(prometheus.NewRegistry())

	testUser := &models.Identity{Name: "test"}

	err := data.CreateIdentity(s.db, testUser)
	assert.NilError(t, err)

	internalProvider := data.InfraProvider(s.db)

	testAccessKey, err := data.CreateAccessKey(s.db, &models.AccessKey{
		Name:       "test",
		IssuedFor:  testUser.ID,
		ExpiresAt:  time.Now().Add(time.Hour),
		ProviderID: internalProvider.ID,
	})
	assert.NilError(t, err)

	route := fmt.Sprintf("/api/users/%s", testUser.ID)
	req, err := http.NewRequest(http.MethodDelete, route, nil)
	assert.NilError(t, err)

	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", testAccessKey))

	resp := httptest.NewRecorder()
	routes.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusBadRequest, resp.Code, resp.Body.String())
}

func TestAPI_CreateGrant_Success(t *testing.T) {
	srv := setupServer(t, withAdminUser)
	routes := srv.GenerateRoutes(prometheus.NewRegistry())

	reqBody := strings.NewReader(`
		{
		  "user": "TJ",
		  "privilege": "admin-role",
		  "resource": "some-cluster"
		}`)

	resp := httptest.NewRecorder()
	req, err := http.NewRequest(http.MethodPost, "/api/grants", reqBody)
	assert.NilError(t, err)
	req.Header.Add("Authorization", "Bearer "+adminAccessKey(srv))
	req.Header.Add("Infra-Version", "0.12.3")

	accessKey, err := data.ValidateAccessKey(srv.db, adminAccessKey(srv))
	assert.NilError(t, err)

	runStep(t, "full JSON response", func(t *testing.T) {
		routes.ServeHTTP(resp, req)
		assert.Equal(t, resp.Code, http.StatusCreated)

		expected := jsonUnmarshal(t, fmt.Sprintf(`
		{
		  "id": "<any-valid-uid>",
		  "created_by": "%[1]v",
		  "privilege": "admin-role",
		  "resource": "some-cluster",
		  "user": "TJ",
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
		req, err := http.NewRequest(http.MethodGet, "/api/grants/"+newGrant.ID.String(), nil)
		assert.NilError(t, err)
		req.Header.Add("Authorization", "Bearer "+adminAccessKey(srv))
		req.Header.Add("Infra-Version", "0.12.3")

		routes.ServeHTTP(resp, req)
		assert.Equal(t, resp.Code, http.StatusOK)

		var getGrant api.Grant
		err = json.NewDecoder(resp.Body).Decode(&getGrant)
		assert.NilError(t, err)
		assert.DeepEqual(t, getGrant, newGrant)
	})
}

func TestAPI_CreateGrantV0_12_2_Success(t *testing.T) {
	srv := setupServer(t, withAdminUser)
	routes := srv.GenerateRoutes(prometheus.NewRegistry())

	reqBody := strings.NewReader(`
		{
		  "subject": "i:TJ",
		  "privilege": "admin-role",
		  "resource": "some-cluster"
		}`)

	resp := httptest.NewRecorder()
	req, err := http.NewRequest(http.MethodPost, "/v1/grants", reqBody)
	assert.NilError(t, err)
	req.Header.Add("Authorization", "Bearer "+adminAccessKey(srv))
	req.Header.Add("Infra-Version", "0.12.2")

	accessKey, err := data.ValidateAccessKey(srv.db, adminAccessKey(srv))
	assert.NilError(t, err)

	runStep(t, "full JSON response", func(t *testing.T) {
		routes.ServeHTTP(resp, req)
		assert.Equal(t, resp.Code, http.StatusCreated)

		expected := jsonUnmarshal(t, fmt.Sprintf(`
		{
		  "id": "<any-valid-uid>",
		  "created_by": "%[1]v",
		  "privilege": "admin-role",
		  "resource": "some-cluster",
		  "subject": "i:TJ",
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
		req.Header.Add("Infra-Version", "0.12.2")

		routes.ServeHTTP(resp, req)
		assert.Equal(t, resp.Code, http.StatusOK)

		var getGrant api.Grant
		err = json.NewDecoder(resp.Body).Decode(&getGrant)
		assert.NilError(t, err)
		assert.DeepEqual(t, getGrant, newGrant)
	})
}

func TestAPI_ListGrantsV0_12_2(t *testing.T) {
	srv := setupServer(t, withAdminUser)
	routes := srv.GenerateRoutes(prometheus.NewRegistry())

	resp := httptest.NewRecorder()
	req, err := http.NewRequest(http.MethodGet, "/v1/grants?privilege=admin", nil)
	assert.NilError(t, err)
	req.Header.Add("Authorization", "Bearer "+adminAccessKey(srv))
	req.Header.Add("Infra-Version", "0.12.2")

	routes.ServeHTTP(resp, req)
	assert.Equal(t, resp.Code, http.StatusOK)

	admin, err := data.ListIdentities(srv.db, &models.Pagination{}, data.ByName("admin@example.com"))
	assert.NilError(t, err)

	expected := jsonUnmarshal(t, fmt.Sprintf(`
	[
		{
			"id": "<any-valid-uid>",
			"created_by": "%[1]v",
			"privilege": "admin",
			"resource": "infra",
			"subject": "%[2]v",
			"created": "%[3]v",
			"updated": "%[3]v"
		}
	]`,
		uid.ID(1),
		uid.NewIdentityPolymorphicID(admin[0].ID),
		time.Now().UTC().Format(time.RFC3339),
	))

	actual := jsonUnmarshal(t, resp.Body.String())
	assert.NilError(t, err)
	assert.DeepEqual(t, actual, expected, cmpAPIGrantJSON)
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

func TestAPI_GetUser(t *testing.T) {
	srv := setupServer(t, withAdminUser)
	routes := srv.GenerateRoutes(prometheus.NewRegistry())

	createID := func(t *testing.T, name string) uid.ID {
		t.Helper()
		var buf bytes.Buffer
		body := api.CreateUserRequest{Name: name}
		err := json.NewEncoder(&buf).Encode(body)
		assert.NilError(t, err)

		req, err := http.NewRequest(http.MethodPost, "/api/users", &buf)
		assert.NilError(t, err)
		req.Header.Add("Authorization", "Bearer "+adminAccessKey(srv))

		resp := httptest.NewRecorder()
		routes.ServeHTTP(resp, req)
		assert.Equal(t, resp.Code, http.StatusCreated, resp.Body.String())
		respObj := &api.CreateUserResponse{}
		err = json.Unmarshal(resp.Body.Bytes(), respObj)
		assert.NilError(t, err)
		return respObj.ID
	}
	idMe := createID(t, "me@example.com")
	idHal := createID(t, "HAL@example.com")

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
			urlPath: "/api/users/" + idMe.String(),
			setup: func(t *testing.T, req *http.Request) {
				req.Header.Del("Authorization")
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusUnauthorized)
			},
		},
		"not authorized": {
			urlPath: "/api/users/" + idHal.String(),
			setup: func(t *testing.T, req *http.Request) {
				key, _ := createAccessKey(t, srv.db, "someonenew@example.com")

				req.Header.Set("Authorization", "Bearer "+key)
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusForbidden)
			},
		},
		"identity not found": {
			urlPath: "/api/users/2341",
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusNotFound)
			},
		},
		"identity by ID for self": {
			urlPath: "/api/users/" + idMe.String(),
			setup: func(t *testing.T, req *http.Request) {
				req.Header.Set("Authorization", "Bearer "+accessKeyMe)
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusOK)
			},
		},
		"identity by ID for someone else": {
			urlPath: "/api/users/" + idMe.String(),
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusOK)
			},
		},
		"identity by self": {
			urlPath: "/api/users/self",
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

				idResponse := api.User{}
				err := json.NewDecoder(resp.Body).Decode(&idResponse)
				assert.NilError(t, err)
				assert.Equal(t, idResponse.ID, idMe)
			},
		},
		"full JSON response": {
			urlPath: "/api/users/" + idMe.String(),
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
						"providerNames": ["infra"],
						"updated": "%[2]v"
					}`,
					idMe.String(),
					time.Now().UTC().Format(time.RFC3339),
				))
				actual := jsonUnmarshal(t, resp.Body.String())
				assert.DeepEqual(t, actual, expected, cmpAPIUserJSON)
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			run(t, tc)
		})
	}
}

var cmpAPIUserJSON = gocmp.Options{
	gocmp.FilterPath(pathMapKey(`created`, `updated`, `lastSeenAt`), cmpApproximateTime),
	gocmp.FilterPath(pathMapKey(`id`), cmpAnyValidUID),
}

func TestAPI_CreateAccessKey(t *testing.T) {
	srv := setupServer(t, withAdminUser)
	routes := srv.GenerateRoutes(prometheus.NewRegistry())

	connector := data.InfraConnectorIdentity(srv.db)

	run := func(body api.CreateAccessKeyRequest) *api.CreateAccessKeyResponse {
		var buf bytes.Buffer
		err := json.NewEncoder(&buf).Encode(body)
		assert.NilError(t, err)

		req, err := http.NewRequest(http.MethodPost, "/api/access-keys", &buf)
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
			UserID:            connector.ID,
			TTL:               api.Duration(time.Minute),
			ExtensionDeadline: api.Duration(time.Minute),
		}

		resp := run(req)
		assert.Assert(t, strings.HasPrefix(resp.Name, "connector-"))
	})

	t.Run("UserProvidedName", func(t *testing.T) {
		req := api.CreateAccessKeyRequest{
			UserID:            connector.ID,
			Name:              "mysupersecretaccesskey",
			TTL:               api.Duration(time.Minute),
			ExtensionDeadline: api.Duration(time.Minute),
		}

		resp := run(req)
		assert.Equal(t, resp.Name, "mysupersecretaccesskey")
	})
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

func TestAPI_DeleteGrant(t *testing.T) {
	srv := setupServer(t, withAdminUser)
	routes := srv.GenerateRoutes(prometheus.NewRegistry())

	user := &models.Identity{Name: "non-admin"}

	err := data.CreateIdentity(srv.db, user)
	assert.NilError(t, err)

	t.Run("last infra admin is deleted", func(t *testing.T) {
		infraAdminGrants, err := data.ListGrants(srv.db, &models.Pagination{}, data.ByPrivilege(models.InfraAdminRole), data.ByResource("infra"))
		assert.NilError(t, err)
		assert.Assert(t, len(infraAdminGrants) == 1)

		req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/grants/%s", infraAdminGrants[0].ID), nil)
		req.Header.Set("Authorization", "Bearer "+adminAccessKey(srv))

		resp := httptest.NewRecorder()
		routes.ServeHTTP(resp, req)

		assert.Equal(t, resp.Code, http.StatusBadRequest, resp.Body.String())
	})

	t.Run("not last infra admin is deleted", func(t *testing.T) {
		grant2 := &models.Grant{
			Subject:   uid.NewIdentityPolymorphicID(user.ID),
			Privilege: models.InfraAdminRole,
			Resource:  "infra",
		}

		err := data.CreateGrant(srv.db, grant2)
		assert.NilError(t, err)

		req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/grants/%s", grant2.ID), nil)
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

		req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/grants/%s", grant2.ID), nil)
		req.Header.Set("Authorization", "Bearer "+adminAccessKey(srv))

		resp := httptest.NewRecorder()
		routes.ServeHTTP(resp, req)

		assert.Equal(t, resp.Code, http.StatusNoContent, resp.Body.String())
	})

	t.Run("last non-infra admin is deleted", func(t *testing.T) {
		grant2 := &models.Grant{
			Subject:   uid.NewIdentityPolymorphicID(user.ID),
			Privilege: "admin",
			Resource:  "example",
		}

		err := data.CreateGrant(srv.db, grant2)
		assert.NilError(t, err)

		req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/grants/%s", grant2.ID), nil)
		req.Header.Set("Authorization", "Bearer "+adminAccessKey(srv))

		resp := httptest.NewRecorder()
		routes.ServeHTTP(resp, req)

		assert.Equal(t, resp.Code, http.StatusNoContent, resp.Body.String())
	})
}

func TestAPI_CreateDestination(t *testing.T) {
	srv := setupServer(t, withAdminUser)
	routes := srv.GenerateRoutes(prometheus.NewRegistry())

	t.Run("does not trim trailing newline from CA", func(t *testing.T) {
		createReq := &api.CreateDestinationRequest{
			Name: "final",
			Connection: api.DestinationConnection{
				URL: "cluster.production.example",
				CA:  "-----BEGIN CERTIFICATE-----\nok\n-----END CERTIFICATE-----\n",
			},
			Resources: []string{"res1", "res2"},
			Roles:     []string{"role1", "role2"},
		}

		body := jsonBody(t, createReq)
		req := httptest.NewRequest(http.MethodPost, "/api/destinations", body)
		req.Header.Set("Authorization", "Bearer "+adminAccessKey(srv))

		resp := httptest.NewRecorder()
		routes.ServeHTTP(resp, req)

		assert.Equal(t, resp.Code, http.StatusCreated, resp.Body.String())

		expected := jsonUnmarshal(t, fmt.Sprintf(`
{
    "id": "<any-valid-uid>",
	"name": "final",
    "uniqueID": "",
    "connection": {
		"url": "cluster.production.example",
		"ca": "-----BEGIN CERTIFICATE-----\nok\n-----END CERTIFICATE-----\n"
	},
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
	})
}

var cmpAPIDestinationJSON = gocmp.Options{
	gocmp.FilterPath(pathMapKey(`created`, `updated`), cmpApproximateTime),
	gocmp.FilterPath(pathMapKey(`id`), cmpAnyValidUID),
}

func TestAPI_LoginResponse(t *testing.T) {
	srv := setupServer(t, withAdminUser)
	routes := srv.GenerateRoutes(prometheus.NewRegistry())

	// setup user to login as
	user := &models.Identity{Name: "steve"}
	err := data.CreateIdentity(srv.db, user)
	assert.NilError(t, err)

	p := data.InfraProvider(srv.db)

	_, err = data.CreateProviderUser(srv.db, p, user)
	assert.NilError(t, err)

	hash, err := bcrypt.GenerateFromPassword([]byte("hunter2"), bcrypt.MinCost)
	assert.NilError(t, err)

	userCredential := &models.Credential{
		IdentityID:   user.ID,
		PasswordHash: hash,
	}

	err = data.CreateCredential(srv.db, userCredential)
	assert.NilError(t, err)

	// do the login request
	loginReq := api.LoginRequest{PasswordCredentials: &api.LoginRequestPasswordCredentials{Name: "steve", Password: "hunter2"}}
	body := jsonBody(t, loginReq)
	req := httptest.NewRequest(http.MethodPost, "/api/login", body)
	req.Header.Add("Infra-Version", "0.13.3")

	resp := httptest.NewRecorder()
	routes.ServeHTTP(resp, req)

	t.Log(resp.Body.String())
	assert.Equal(t, 201, resp.Code)

	loginResp := &api.LoginResponse{}

	err = json.Unmarshal(resp.Body.Bytes(), loginResp)
	assert.NilError(t, err)

	assert.Assert(t, loginResp.AccessKey != "")
	assert.Equal(t, len(resp.Result().Cookies()), 2)

	cookies := make(map[string]string)
	for _, c := range resp.Result().Cookies() {
		cookies[c.Name] = c.Value
	}

	assert.Equal(t, cookies["login"], "1")
	assert.Equal(t, cookies["auth"], loginResp.AccessKey) // make sure the cookie matches the response
	assert.Equal(t, loginResp.UserID, user.ID)
	assert.Equal(t, loginResp.Name, "steve")
	assert.Equal(t, loginResp.PasswordUpdateRequired, false)
}
