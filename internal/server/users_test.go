package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	gocmp "github.com/google/go-cmp/cmp"
	"github.com/prometheus/client_golang/prometheus"
	"gotest.tools/v3/assert"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal/generate"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

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
				assert.Equal(t, resp.Code, http.StatusBadRequest, resp.Body.String())

				respBody := &api.Error{}
				err := json.Unmarshal(resp.Body.Bytes(), respBody)
				assert.NilError(t, err)

				expected := []api.FieldError{
					{FieldName: "limit", Errors: []string{"value (1001) must be at most 1000"}},
				}
				assert.DeepEqual(t, respBody.FieldErrors, expected)
			},
		},
		"invalid page": {
			urlPath: "/api/users?page=-1",
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusBadRequest, resp.Body.String())

				respBody := &api.Error{}
				err := json.Unmarshal(resp.Body.Bytes(), respBody)
				assert.NilError(t, err)

				expected := []api.FieldError{
					{FieldName: "page", Errors: []string{"value (-1) must be at least 0"}},
				}
				assert.DeepEqual(t, respBody.FieldErrors, expected)
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

func TestAPI_CreateUser(t *testing.T) {
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
func TestAPI_CreateUserAndUpdatePassword(t *testing.T) {
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

func TestAPI_DeleteUser(t *testing.T) {
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

func TestAPI_DeleteUser_NoDeleteInternalIdentities(t *testing.T) {
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

func TestAPI_DeleteUser_NoDeleteSelf(t *testing.T) {
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
