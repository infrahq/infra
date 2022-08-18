package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sort"
	"testing"
	"time"

	gocmp "github.com/google/go-cmp/cmp"
	"github.com/prometheus/client_golang/prometheus"
	"gotest.tools/v3/assert"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

func TestAPI_ListGrants(t *testing.T) {
	srv := setupServer(t, withAdminUser)
	routes := srv.GenerateRoutes()

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

	createGrant := func(t *testing.T, user uid.ID, privilege string) {
		t.Helper()
		var buf bytes.Buffer
		body := api.CreateGrantRequest{
			User:      user,
			Privilege: privilege,
			Resource:  "res1",
		}
		err := json.NewEncoder(&buf).Encode(body)
		assert.NilError(t, err)

		// nolint:noctx
		req, err := http.NewRequest(http.MethodPost, "/api/grants", &buf)
		assert.NilError(t, err)
		req.Header.Add("Authorization", "Bearer "+adminAccessKey(srv))
		req.Header.Add("Infra-Version", "0.12.3")

		resp := httptest.NewRecorder()
		routes.ServeHTTP(resp, req)
		assert.Equal(t, resp.Code, http.StatusCreated, resp.Body.String())
	}

	createGroup := func(t *testing.T, name string, users ...uid.ID) uid.ID {
		t.Helper()
		group := &models.Group{Name: name}

		err := data.CreateGroup(srv.DB(), group)
		assert.NilError(t, err)

		err = data.AddUsersToGroup(srv.DB(), group.ID, users)
		assert.NilError(t, err)

		return group.ID
	}

	idInGroup := createID(t, "inagroup@example.com")
	idOther := createID(t, "other@example.com")

	createGrant(t, idInGroup, "custom1")
	createGrant(t, idOther, "custom2")
	createGrant(t, idOther, "connector")

	groupID := createGroup(t, "humans", idInGroup)
	otherGroup := createGroup(t, "others", idOther)

	token := &models.AccessKey{
		IssuedFor:  idInGroup,
		ProviderID: data.InfraProvider(srv.DB()).ID,
		ExpiresAt:  time.Now().Add(10 * time.Minute),
	}

	accessKey, err := data.CreateAccessKey(srv.DB(), token)
	assert.NilError(t, err)

	admin, err := data.GetIdentity(srv.DB(), data.ByName("admin@example.com"))
	assert.NilError(t, err)

	type testCase struct {
		urlPath  string
		setup    func(t *testing.T, req *http.Request)
		expected func(t *testing.T, resp *httptest.ResponseRecorder)
	}

	run := func(t *testing.T, tc testCase) {
		req, err := http.NewRequest(http.MethodGet, tc.urlPath, nil)
		assert.NilError(t, err)
		req.Header.Set("Authorization", "Bearer "+accessKey)
		req.Header.Add("Infra-Version", "0.12.3")

		if tc.setup != nil {
			tc.setup(t, req)
		}

		resp := httptest.NewRecorder()
		routes.ServeHTTP(resp, req)

		tc.expected(t, resp)
	}

	testCases := map[string]testCase{
		"not authenticated": {
			urlPath: "/api/grants?user=" + idOther.String(),
			setup: func(t *testing.T, req *http.Request) {
				req.Header.Del("Authorization")
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusUnauthorized)
			},
		},
		"not authorized, wrong identity": {
			urlPath: "/api/grants?user=" + idOther.String(),
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusForbidden, resp.Body.String())
			},
		},
		"not authorized, wrong group": {
			urlPath: "/api/grants?group=" + otherGroup.String(),
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusForbidden, resp.Body.String())
			},
		},
		"not authorized, no subject in query": {
			urlPath: "/api/grants?resource=res1",
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusForbidden, resp.Body.String())
			},
		},
		"authorized by grant": {
			urlPath: "/api/grants?resource=none",
			setup: func(t *testing.T, req *http.Request) {
				req.Header.Set("Authorization", "Bearer "+adminAccessKey(srv))
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusOK, resp.Body.String())
				var grants api.ListResponse[api.Grant]
				err := json.NewDecoder(resp.Body).Decode(&grants)
				assert.NilError(t, err)
				assert.Equal(t, len(grants.Items), 0) // no grants for this resource
			},
		},
		"authorized by identity matching subject": {
			urlPath: "/api/grants?user=" + idInGroup.String(),
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusOK, resp.Body.String())
				var grants api.ListResponse[api.Grant]
				err := json.NewDecoder(resp.Body).Decode(&grants)
				assert.NilError(t, err)
				expected := []api.Grant{
					{
						User:      idInGroup,
						Privilege: "custom1",
						Resource:  "res1",
					},
				}
				assert.DeepEqual(t, grants.Items, expected, cmpAPIGrantShallow)
			},
		},
		"authorized by group matching subject": {
			urlPath: "/api/grants?group=" + groupID.String(),
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusOK, resp.Body.String())
				var grants api.ListResponse[api.Grant]
				err := json.NewDecoder(resp.Body).Decode(&grants)
				assert.NilError(t, err)
				// no grants for this group
				assert.Equal(t, len(grants.Items), 0)
			},
		},
		"bad request, user and group": {
			urlPath: "/api/grants?group=groupa&user=userB",
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusBadRequest, resp.Body.String())

				respBody := &api.Error{}
				err := json.Unmarshal(resp.Body.Bytes(), respBody)
				assert.NilError(t, err)

				expected := []api.FieldError{
					{Errors: []string{"only one of (user, group) can have a value"}},
				}
				assert.DeepEqual(t, respBody.FieldErrors, expected)
			},
		},
		"no filters": {
			urlPath: "/api/grants?showSystem=true",
			setup: func(t *testing.T, req *http.Request) {
				req.Header.Set("Authorization", "Bearer "+adminAccessKey(srv))
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				connector, err := data.GetIdentity(srv.DB(), data.ByName("connector"))
				assert.NilError(t, err)

				assert.Equal(t, resp.Code, http.StatusOK, resp.Body.String())
				var grants api.ListResponse[api.Grant]
				err = json.NewDecoder(resp.Body).Decode(&grants)
				assert.NilError(t, err)

				expected := []api.Grant{
					{
						User:      admin.ID,
						Privilege: "admin",
						Resource:  "infra",
					},
					{
						User:      connector.ID,
						Privilege: "connector",
						Resource:  "infra",
					},
					{
						User:      idInGroup,
						Privilege: "custom1",
						Resource:  "res1",
					},
					{
						User:      idOther,
						Privilege: "custom2",
						Resource:  "res1",
					},
					{
						User:      idOther,
						Privilege: "connector",
						Resource:  "res1",
					},
				}
				// check sort
				assert.Assert(t, sort.SliceIsSorted(grants.Items, func(i, j int) bool {
					return grants.Items[i].ID < grants.Items[j].ID
				}))
				assert.DeepEqual(t, grants.Items, expected, cmpAPIGrantShallow)
			},
		},
		"no filter, page 2": {
			urlPath: "/api/grants?page=2&limit=2&showSystem=true",
			setup: func(t *testing.T, req *http.Request) {
				req.Header.Set("Authorization", "Bearer "+adminAccessKey(srv))
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusOK, resp.Body.String())
				var grants api.ListResponse[api.Grant]
				err = json.NewDecoder(resp.Body).Decode(&grants)
				assert.NilError(t, err)

				expected := []api.Grant{
					{
						User:      idInGroup,
						Privilege: "custom1",
						Resource:  "res1",
					},
					{
						User:      idOther,
						Privilege: "custom2",
						Resource:  "res1",
					},
				}
				assert.DeepEqual(t, grants.Items, expected, cmpAPIGrantShallow)
				assert.Equal(t, grants.PaginationResponse, api.PaginationResponse{Limit: 2, Page: 2, TotalCount: 5, TotalPages: 3})
			},
		},
		"hide infra connector": {
			urlPath: "/api/grants",
			setup: func(t *testing.T, req *http.Request) {
				req.Header.Set("Authorization", "Bearer "+adminAccessKey(srv))
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusOK, resp.Body.String())
				var grants api.ListResponse[api.Grant]
				err = json.NewDecoder(resp.Body).Decode(&grants)
				assert.NilError(t, err)

				expected := []api.Grant{
					{
						User:      admin.ID,
						Privilege: "admin",
						Resource:  "infra",
					},
					{
						User:      idInGroup,
						Privilege: "custom1",
						Resource:  "res1",
					},
					{
						User:      idOther,
						Privilege: "custom2",
						Resource:  "res1",
					},
					{
						User:      idOther,
						Privilege: "connector",
						Resource:  "res1",
					},
				}
				// check sort
				assert.Assert(t, sort.SliceIsSorted(grants.Items, func(i, j int) bool {
					return grants.Items[i].ID < grants.Items[j].ID
				}))
				assert.DeepEqual(t, grants.Items, expected, cmpAPIGrantShallow)
			},
		},
		"filter by resource": {
			urlPath: "/api/grants?resource=res1",
			setup: func(t *testing.T, req *http.Request) {
				req.Header.Set("Authorization", "Bearer "+adminAccessKey(srv))
				req.Header.Add("Infra-Version", "0.12.3")
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusOK, resp.Body.String())
				var grants api.ListResponse[api.Grant]
				err = json.NewDecoder(resp.Body).Decode(&grants)
				assert.NilError(t, err)

				expected := []api.Grant{
					{
						User:      idInGroup,
						Privilege: "custom1",
						Resource:  "res1",
					},
					{
						User:      idOther,
						Privilege: "custom2",
						Resource:  "res1",
					},
					{
						User:      idOther,
						Privilege: "connector",
						Resource:  "res1",
					},
				}
				assert.DeepEqual(t, grants.Items, expected, cmpAPIGrantShallow)
			},
		},
		"filter by resource and privilege": {
			urlPath: "/api/grants?resource=res1&privilege=custom1",
			setup: func(t *testing.T, req *http.Request) {
				req.Header.Set("Authorization", "Bearer "+adminAccessKey(srv))
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusOK, resp.Body.String())
				var grants api.ListResponse[api.Grant]
				err = json.NewDecoder(resp.Body).Decode(&grants)
				assert.NilError(t, err)

				expected := []api.Grant{
					{
						User:      idInGroup,
						Privilege: "custom1",
						Resource:  "res1",
					},
				}
				assert.DeepEqual(t, grants.Items, expected, cmpAPIGrantShallow)
			},
		},
		"full JSON response": {
			urlPath: "/api/grants?user=" + idInGroup.String(),
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusOK, resp.Body.String())

				expected := jsonUnmarshal(t, fmt.Sprintf(`
					{
						"count": 1,
						"limit": 100,
						"page": 1,
						"totalPages": 1,
						"totalCount": 1,
						"items": [{
							"id": "<any-valid-uid>",
							"created_by": "%[1]v",
							"privilege": "custom1",
							"resource": "res1",
							"user": "%[2]v",
							"created": "%[3]v",
							"updated": "%[3]v"
						}]
					}`,
					admin.ID,
					idInGroup.String(),
					time.Now().UTC().Format(time.RFC3339),
				))
				actual := jsonUnmarshal(t, resp.Body.String())
				assert.DeepEqual(t, actual, expected, cmpAPIGrantJSON)
			},
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			run(t, tc)
		})
	}
}

func TestAPI_ListGrants_InheritedGrants(t *testing.T) {
	srv := setupServer(t, withAdminUser)
	routes := srv.GenerateRoutes()

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

	createGroup := func(t *testing.T, name string, users ...uid.ID) uid.ID {
		t.Helper()
		group := &models.Group{Name: name}

		err := data.CreateGroup(srv.DB(), group)
		assert.NilError(t, err)

		err = data.AddUsersToGroup(srv.DB(), group.ID, users)
		assert.NilError(t, err)

		return group.ID
	}

	idInGroup := createID(t, "inagroup@example.com")
	mikhail := createID(t, "mikhail@example.com")

	zoologistsID := createGroup(t, "Zoologists", mikhail)

	loginAs := func(t *testing.T, userID uid.ID, req *http.Request) {
		t.Helper()
		token := &models.AccessKey{
			IssuedFor:  userID,
			ProviderID: data.InfraProvider(srv.DB()).ID,
			ExpiresAt:  time.Now().Add(10 * time.Minute),
		}

		var err error
		accessKey, err := data.CreateAccessKey(srv.DB(), token)
		assert.NilError(t, err)

		req.Header.Set("Authorization", "Bearer "+accessKey)
	}

	err := data.CreateGrant(srv.DB(), &models.Grant{
		Resource:  "infra",
		Privilege: "view",
		Subject:   uid.NewIdentityPolymorphicID(idInGroup),
	})
	assert.NilError(t, err)

	err = data.CreateGrant(srv.DB(), &models.Grant{
		Subject:   uid.NewGroupPolymorphicID(zoologistsID),
		Privilege: "examine",
		Resource:  "butterflies",
	})
	assert.NilError(t, err)

	type testCase struct {
		urlPath  string
		setup    func(t *testing.T, req *http.Request)
		expected func(t *testing.T, resp *httptest.ResponseRecorder)
	}

	run := func(t *testing.T, tc testCase) {
		req, err := http.NewRequest(http.MethodGet, tc.urlPath, nil)
		assert.NilError(t, err)

		req.Header.Add("Infra-Version", "0.12.3")

		if tc.setup != nil {
			tc.setup(t, req)
		}

		resp := httptest.NewRecorder()
		routes.ServeHTTP(resp, req)

		tc.expected(t, resp)
	}

	testCases := map[string]testCase{
		"authorized by inherited group matching subject": {
			urlPath: "/api/grants?resource=butterflies&showInherited=1&user=" + mikhail.String(),
			setup: func(t *testing.T, req *http.Request) {
				loginAs(t, idInGroup, req)
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusOK, resp.Body.String())
				var grants api.ListResponse[api.Grant]
				err := json.NewDecoder(resp.Body).Decode(&grants)
				assert.NilError(t, err)
				expected := []api.Grant{
					{
						Group:     zoologistsID,
						Privilege: "examine",
						Resource:  "butterflies",
					},
				}
				assert.DeepEqual(t, grants.Items, expected, cmpAPIGrantShallow)
			},
		},
		"can list grants without a subject": {
			urlPath: "/api/grants?showInherited=1&resource=dinosaurs", // inherited doesn't mean anything here
			setup: func(t *testing.T, req *http.Request) {
				loginAs(t, idInGroup, req)

				err = data.CreateGrant(srv.DB(), &models.Grant{
					Subject:   uid.NewGroupPolymorphicID(zoologistsID),
					Privilege: "examine",
					Resource:  "dinosaurs",
				})
				assert.NilError(t, err)
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusOK, resp.Body.String())
				var grants api.ListResponse[api.Grant]
				err := json.NewDecoder(resp.Body).Decode(&grants)
				assert.NilError(t, err)
				expected := []api.Grant{
					{
						Group:     zoologistsID,
						Privilege: "examine",
						Resource:  "dinosaurs",
					},
				}
				assert.DeepEqual(t, grants.Items, expected, cmpAPIGrantShallow)
			},
		},
		"user can select grants for groups they are a member of": {
			urlPath: "/api/grants?resource=butterflies&group=" + zoologistsID.String(),
			setup: func(t *testing.T, req *http.Request) {
				loginAs(t, mikhail, req)
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusOK)
				var grants api.ListResponse[api.Grant]
				err := json.NewDecoder(resp.Body).Decode(&grants)
				assert.NilError(t, err)
				expected := []api.Grant{
					{
						Group:     zoologistsID,
						Privilege: "examine",
						Resource:  "butterflies",
					},
				}
				assert.DeepEqual(t, grants.Items, expected, cmpAPIGrantShallow)
			},
		},
		"user can select their own inherited grants without any special permissions": {
			urlPath: "/api/grants?showInherited=1&resource=butterflies&user=" + mikhail.String(),
			setup: func(t *testing.T, req *http.Request) {
				loginAs(t, mikhail, req)
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusOK, resp.Body.String())
				var grants api.ListResponse[api.Grant]
				err = json.NewDecoder(resp.Body).Decode(&grants)
				assert.NilError(t, err)
				expected := []api.Grant{
					{
						Group:     zoologistsID,
						Privilege: "examine",
						Resource:  "butterflies",
					},
				}
				assert.DeepEqual(t, grants.Items, expected, cmpAPIGrantShallow)
			},
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			run(t, tc)
		})
	}
}

var cmpAPIGrantShallow = gocmp.Comparer(func(x, y api.Grant) bool {
	return x.User == y.User && x.Privilege == y.Privilege && x.Resource == y.Resource
})

var cmpAPIGrantJSON = gocmp.Options{
	gocmp.FilterPath(pathMapKey(`created`, `updated`), cmpApproximateTime),
	gocmp.FilterPath(pathMapKey(`id`), cmpAnyValidUID),
}

func TestAPI_CreateGrant(t *testing.T) {
	srv := setupServer(t, withAdminUser)
	routes := srv.GenerateRoutes()

	accessKey, err := data.ValidateAccessKey(srv.DB(), adminAccessKey(srv))
	assert.NilError(t, err)

	someUser := models.Identity{Name: "someone@example.com"}
	err = data.CreateIdentity(srv.DB(), &someUser)
	assert.NilError(t, err)

	supportAdmin := models.Identity{Name: "support-admin@example.com"}
	err = data.CreateIdentity(srv.DB(), &supportAdmin)
	assert.NilError(t, err)

	supportAdminGrant := models.Grant{
		Subject:   supportAdmin.PolyID(),
		Privilege: models.InfraSupportAdminRole,
		Resource:  "infra",
	}
	err = data.CreateGrant(srv.DB(), &supportAdminGrant)
	assert.NilError(t, err)

	token := &models.AccessKey{
		IssuedFor:  supportAdmin.ID,
		ProviderID: data.InfraProvider(srv.DB()).ID,
		ExpiresAt:  time.Now().Add(10 * time.Second),
	}

	supportAccessKeyStr, err := data.CreateAccessKey(srv.DB(), token)
	assert.NilError(t, err)

	type testCase struct {
		setup    func(t *testing.T, req *http.Request)
		expected func(t *testing.T, resp *httptest.ResponseRecorder)
		body     api.CreateGrantRequest
	}

	run := func(t *testing.T, tc testCase) {
		body := jsonBody(t, tc.body)
		req, err := http.NewRequest(http.MethodPost, "/api/grants", body)
		assert.NilError(t, err)
		req.Header.Add("Infra-Version", "0.12.3")

		if tc.setup != nil {
			tc.setup(t, req)
		}

		resp := httptest.NewRecorder()
		routes.ServeHTTP(resp, req)

		tc.expected(t, resp)
	}

	testCases := map[string]testCase{
		"missing required fields": {
			setup: func(t *testing.T, req *http.Request) {
				req.Header.Set("Authorization", "Bearer "+adminAccessKey(srv))
			},
			body: api.CreateGrantRequest{},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusBadRequest, resp.Body.String())

				respBody := &api.Error{}
				err := json.Unmarshal(resp.Body.Bytes(), respBody)
				assert.NilError(t, err)

				expected := []api.FieldError{
					{Errors: []string{"one of (user, group) is required"}},
					{FieldName: "privilege", Errors: []string{"is required"}},
					{FieldName: "resource", Errors: []string{"is required"}},
				}
				assert.DeepEqual(t, respBody.FieldErrors, expected)
			},
		},
		"success": {
			setup: func(t *testing.T, req *http.Request) {
				req.Header.Set("Authorization", "Bearer "+adminAccessKey(srv))
			},
			body: api.CreateGrantRequest{
				User:      someUser.ID,
				Privilege: models.InfraAdminRole,
				Resource:  "some-cluster",
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusCreated)

				expected := jsonUnmarshal(t, fmt.Sprintf(`
				{
					"id": "<any-valid-uid>",
					"created_by": "%[1]v",
					"privilege": "%[2]v",
					"resource": "some-cluster",
					"user": "%[3]v",
					"created": "%[4]v",
					"updated": "%[4]v",
					"wasCreated": true
				}`,
					accessKey.IssuedFor,
					models.InfraAdminRole,
					someUser.ID.String(),
					time.Now().UTC().Format(time.RFC3339),
				))
				actual := jsonUnmarshal(t, resp.Body.String())
				assert.DeepEqual(t, actual, expected, cmpAPIGrantJSON)
			},
		},
		"admin can not grant infra support admin role": {
			setup: func(t *testing.T, req *http.Request) {
				req.Header.Set("Authorization", "Bearer "+adminAccessKey(srv))
			},
			body: api.CreateGrantRequest{
				User:      someUser.ID,
				Privilege: models.InfraSupportAdminRole,
				Resource:  "infra",
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusForbidden)
			},
		},
		"support admin grant": {
			setup: func(t *testing.T, req *http.Request) {
				req.Header.Set("Authorization", "Bearer "+supportAccessKeyStr)
			},
			body: api.CreateGrantRequest{
				User:      someUser.ID,
				Privilege: models.InfraSupportAdminRole,
				Resource:  "infra",
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusCreated)

				expected := jsonUnmarshal(t, fmt.Sprintf(`
				{
					"id": "<any-valid-uid>",
					"created_by": "%[1]v",
					"privilege": "%[2]v",
					"resource": "infra",
					"user": "%[3]v",
					"created": "%[4]v",
					"updated": "%[4]v",
					"wasCreated": true
				}`,
					supportAdmin.ID,
					models.InfraSupportAdminRole,
					someUser.ID.String(),
					time.Now().UTC().Format(time.RFC3339),
				))
				actual := jsonUnmarshal(t, resp.Body.String())
				assert.DeepEqual(t, actual, expected, cmpAPIGrantJSON)
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			run(t, tc)
		})
	}
}

func TestAPI_DeleteGrant(t *testing.T) {
	srv := setupServer(t, withAdminUser)
	routes := srv.GenerateRoutes()

	user := &models.Identity{Name: "non-admin"}

	err := data.CreateIdentity(srv.DB(), user)
	assert.NilError(t, err)

	t.Run("last infra admin is deleted", func(t *testing.T) {
		infraAdminGrants, err := data.ListGrants(srv.DB(), nil, data.ByPrivilege(models.InfraAdminRole), data.ByResource("infra"))
		assert.NilError(t, err)
		assert.Assert(t, len(infraAdminGrants) == 1)

		req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/grants/%s", infraAdminGrants[0].ID), nil)
		req.Header.Set("Authorization", "Bearer "+adminAccessKey(srv))
		req.Header.Set("Infra-Version", apiVersionLatest)

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

		err := data.CreateGrant(srv.DB(), grant2)
		assert.NilError(t, err)

		req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/grants/%s", grant2.ID), nil)
		req.Header.Set("Authorization", "Bearer "+adminAccessKey(srv))
		req.Header.Set("Infra-Version", apiVersionLatest)

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

		err := data.CreateGrant(srv.DB(), grant2)
		assert.NilError(t, err)

		req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/grants/%s", grant2.ID), nil)
		req.Header.Set("Authorization", "Bearer "+adminAccessKey(srv))
		req.Header.Set("Infra-Version", apiVersionLatest)

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

		err := data.CreateGrant(srv.DB(), grant2)
		assert.NilError(t, err)

		req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/grants/%s", grant2.ID), nil)
		req.Header.Set("Authorization", "Bearer "+adminAccessKey(srv))
		req.Header.Set("Infra-Version", apiVersionLatest)

		resp := httptest.NewRecorder()
		routes.ServeHTTP(resp, req)

		assert.Equal(t, resp.Code, http.StatusNoContent, resp.Body.String())
	})
}
