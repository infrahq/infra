package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"
	"time"

	gocmp "github.com/google/go-cmp/cmp"
	"golang.org/x/sync/errgroup"
	"gotest.tools/v3/assert"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

func TestAPI_ListGrants(t *testing.T) {
	srv := setupServer(t, withAdminUser, withMultiOrgEnabled)
	routes := srv.GenerateRoutes()

	createID := func(t *testing.T, name string) uid.ID {
		t.Helper()
		var buf bytes.Buffer
		body := api.CreateUserRequest{Name: name}
		err := json.NewEncoder(&buf).Encode(body)
		assert.NilError(t, err)

		req := httptest.NewRequest(http.MethodPost, "/api/users", &buf)
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

	createGrant := func(t *testing.T, user uid.ID, privilege, resource string) {
		t.Helper()
		var buf bytes.Buffer
		body := api.GrantRequest{
			User:      user,
			Privilege: privilege,
			Resource:  resource,
		}
		err := json.NewEncoder(&buf).Encode(body)
		assert.NilError(t, err)

		// nolint:noctx
		req := httptest.NewRequest(http.MethodPost, "/api/grants", &buf)
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

	createGrant(t, idInGroup, "custom1", "res1")
	createGrant(t, idOther, "custom2", "res1.ns1")
	createGrant(t, idOther, "connector", "res1.ns2")

	groupID := createGroup(t, "humans", idInGroup)
	otherGroup := createGroup(t, "others", idOther)

	token := &models.AccessKey{
		IssuedFor:  idInGroup,
		ProviderID: data.InfraProvider(srv.DB()).ID,
		ExpiresAt:  time.Now().Add(10 * time.Minute),
	}

	accessKey, err := data.CreateAccessKey(srv.DB(), token)
	assert.NilError(t, err)

	admin, err := data.GetIdentity(srv.DB(), data.GetIdentityOptions{ByName: "admin@example.com"})
	assert.NilError(t, err)

	otherOrg := createOtherOrg(t, srv.db)

	type testCase struct {
		urlPath  string
		setup    func(t *testing.T, req *http.Request)
		expected func(t *testing.T, resp *httptest.ResponseRecorder)
	}

	run := func(t *testing.T, tc testCase) {
		req := httptest.NewRequest(http.MethodGet, tc.urlPath, nil)
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
		"not authorized, admin for wrong org": {
			urlPath: "/api/grants?resource=res1",
			setup: func(t *testing.T, req *http.Request) {
				req.Host = "example.com"
				req.Header.Set("Authorization", "Bearer "+otherOrg.AdminAccessKey)
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusBadRequest, resp.Body.String())
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
				connector, err := data.GetIdentity(srv.DB(), data.GetIdentityOptions{ByName: "connector"})
				assert.NilError(t, err)

				assert.Equal(t, resp.Code, http.StatusOK, resp.Body.String())
				var grants api.ListResponse[api.Grant]
				err = json.NewDecoder(resp.Body).Decode(&grants)
				assert.NilError(t, err)

				expected := []api.Grant{
					{
						User:      connector.ID,
						Privilege: "connector",
						Resource:  "infra",
					},
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
						Resource:  "res1.ns1",
					},
					{
						User:      idOther,
						Privilege: "connector",
						Resource:  "res1.ns2",
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
						Resource:  "res1.ns1",
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
						Resource:  "res1.ns1",
					},
					{
						User:      idOther,
						Privilege: "connector",
						Resource:  "res1.ns2",
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
		"filter by destination": {
			urlPath: "/api/grants?destination=res1",
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
						Resource:  "res1.ns1",
					},
					{
						User:      idOther,
						Privilege: "connector",
						Resource:  "res1.ns2",
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
		"unsupported filter with update index": {
			urlPath: "/api/grants?destination=res1&lastUpdateIndex=1&user=1234",
			setup: func(t *testing.T, req *http.Request) {
				req.Header.Set("Authorization", "Bearer "+adminAccessKey(srv))
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusBadRequest, resp.Body.String())

				respBody := &api.Error{}
				err := json.Unmarshal(resp.Body.Bytes(), respBody)
				assert.NilError(t, err)

				expected := []api.FieldError{
					{
						FieldName: "lastUpdateIndex",
						Errors:    []string{"can not be used with user parameter(s)"},
					},
				}
				assert.DeepEqual(t, respBody.FieldErrors, expected)
			},
		},
		"no filter with update index": {
			urlPath: "/api/grants?lastUpdateIndex=1",
			setup: func(t *testing.T, req *http.Request) {
				req.Header.Set("Authorization", "Bearer "+adminAccessKey(srv))
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusBadRequest, resp.Body.String())

				respBody := &api.Error{}
				err := json.Unmarshal(resp.Body.Bytes(), respBody)
				assert.NilError(t, err)

				expected := []api.FieldError{
					{
						FieldName: "lastUpdateIndex",
						Errors:    []string{"requires a supported filter"},
					},
				}
				assert.DeepEqual(t, respBody.FieldErrors, expected)
			},
		},
		"with stale update index": {
			urlPath: "/api/grants?destination=res1&lastUpdateIndex=1",
			setup: func(t *testing.T, req *http.Request) {
				req.Header.Set("Authorization", "Bearer "+adminAccessKey(srv))
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusOK, resp.Body.String())
				var grants api.ListResponse[api.Grant]
				err = json.NewDecoder(resp.Body).Decode(&grants)
				assert.NilError(t, err)

				expected := api.ListResponse[api.Grant]{
					Items: []api.Grant{
						{User: idInGroup, Privilege: "custom1", Resource: "res1"},
						{User: idOther, Privilege: "custom2", Resource: "res1.ns1"},
						{User: idOther, Privilege: "connector", Resource: "res1.ns2"},
					},
					Count: 3,
				}
				assert.DeepEqual(t, grants, expected, cmpAPIGrantShallow)
				assert.Equal(t, resp.Result().Header.Get("Last-Update-Index"), "10004")
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

		req := httptest.NewRequest(http.MethodPost, "/api/users", &buf)
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
		req := httptest.NewRequest(http.MethodGet, tc.urlPath, nil)
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
		"requires a userID with showInherited": {
			urlPath: "/api/grants?showInherited=1&resource=dinosaurs",
			setup: func(t *testing.T, req *http.Request) {
				loginAs(t, idInGroup, req)
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusBadRequest, resp.Body.String())

				var respBody api.Error
				err := json.NewDecoder(resp.Body).Decode(&respBody)
				assert.NilError(t, err)

				expected := []api.FieldError{
					{FieldName: "showInherited", Errors: []string{"requires a user ID"}},
				}
				assert.DeepEqual(t, respBody.FieldErrors, expected)
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

func TestAPI_ListGrants_ExtendedRequestTimeout(t *testing.T) {
	if testing.Short() {
		t.Skip("too long for short run")
	}

	withShortRequestTimeout := func(t *testing.T, options *Options) {
		options.API.RequestTimeout = 250 * time.Millisecond
		options.API.BlockingRequestTimeout = 1500 * time.Millisecond
	}
	srv := setupServer(t, withAdminUser, withShortRequestTimeout)
	routes := srv.GenerateRoutes()

	urlPath := "/api/grants?destination=infra&lastUpdateIndex=10001"
	req := httptest.NewRequest(http.MethodGet, urlPath, nil)
	req.Header.Set("Authorization", "Bearer "+adminAccessKey(srv))
	req.Header.Add("Infra-Version", apiVersionLatest)

	start := time.Now()
	resp := httptest.NewRecorder()
	routes.ServeHTTP(resp, req)

	elapsed := time.Since(start)
	assert.Assert(t, elapsed >= srv.options.API.BlockingRequestTimeout,
		"elapsed=%v %v", elapsed, (*responseDebug)(resp))

	assert.Equal(t, resp.Code, http.StatusNotModified, (*responseDebug)(resp))
}

func TestAPI_ListGrants_ExtendedRequestTimeout_CancelledByClient(t *testing.T) {
	if testing.Short() {
		t.Skip("too long for short run")
	}

	withShortRequestTimeout := func(t *testing.T, options *Options) {
		options.API.RequestTimeout = 250 * time.Millisecond
		options.API.BlockingRequestTimeout = 2500 * time.Millisecond
	}
	srv := setupServer(t, withAdminUser, withShortRequestTimeout)
	routes := srv.GenerateRoutes()

	urlPath := "/api/grants?destination=infra&lastUpdateIndex=10001"
	req := httptest.NewRequest(http.MethodGet, urlPath, nil)
	req.Header.Set("Authorization", "Bearer "+adminAccessKey(srv))
	req.Header.Add("Infra-Version", apiVersionLatest)

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	t.Cleanup(cancel)
	req = req.WithContext(ctx)

	start := time.Now()
	resp := httptest.NewRecorder()
	routes.ServeHTTP(resp, req)

	elapsed := time.Since(start)
	assert.Assert(t, elapsed < time.Second,
		"elapsed=%v %v", elapsed, (*responseDebug)(resp))

	assert.Equal(t, resp.Code, http.StatusNotModified, (*responseDebug)(resp))
}

type responseDebug httptest.ResponseRecorder

func (r *responseDebug) String() string {
	if r == nil {
		return "<nil>"
	}
	resp := (*httptest.ResponseRecorder)(r)
	return fmt.Sprintf("code=%v headers=%v\n%v",
		r.Code,
		resp.Result().Header,
		resp.Body.String())
}

func TestAPI_ListGrants_BlockingRequest_BlocksUntilUpdate(t *testing.T) {
	if testing.Short() {
		t.Skip("too long for short run")
	}

	srv := setupServer(t, withAdminUser)
	routes := srv.GenerateRoutes()

	g := errgroup.Group{}
	respCh := make(chan *httptest.ResponseRecorder)
	g.Go(func() error {
		urlPath := "/api/grants?destination=infra&lastUpdateIndex=10001"
		req := httptest.NewRequest(http.MethodGet, urlPath, nil)
		req.Header.Set("Authorization", "Bearer "+adminAccessKey(srv))
		req.Header.Add("Infra-Version", apiVersionLatest)

		resp := httptest.NewRecorder()
		routes.ServeHTTP(resp, req)
		respCh <- resp
		return nil
	})

	isBlocked(t, respCh)

	// unrelated grant
	err := data.CreateGrant(srv.db, &models.Grant{
		Subject:   "i:abcd",
		Privilege: "view",
		Resource:  "somethingelse",
	})
	assert.NilError(t, err)
	isBlocked(t, respCh)

	// matching grant
	err = data.CreateGrant(srv.db, &models.Grant{
		Subject:   "i:abcd",
		Privilege: "view",
		Resource:  "infra",
	})
	assert.NilError(t, err)

	resp := isNotBlocked(t, respCh)
	assert.Equal(t, resp.Code, http.StatusOK, (*responseDebug)(resp))
	assert.Equal(t, resp.Result().Header.Get("Last-Update-Index"), "10003", (*responseDebug)(resp))

	respBody := &api.ListResponse[api.Grant]{}
	assert.NilError(t, json.NewDecoder(resp.Body).Decode(respBody))

	assert.Equal(t, len(respBody.Items), 2)
}

func isBlocked[T any](t *testing.T, ch chan T) {
	t.Helper()
	select {
	case item := <-ch:
		t.Fatalf("expected request to be blocked, but it returned: %v", item)
	case <-time.After(200 * time.Millisecond):
	}
}

func isNotBlocked[T any](t *testing.T, ch chan T) (result T) {
	t.Helper()
	timeout := 100 * time.Millisecond
	select {
	case item := <-ch:
		return item
	case <-time.After(timeout):
		t.Fatalf("expected request to not block, timeout after: %v", timeout)
		return result
	}
}

func TestAPI_ListGrants_BlockingRequest_NotFoundBlocksUntilUpdate(t *testing.T) {
	if testing.Short() {
		t.Skip("too long for short run")
	}

	srv := setupServer(t, withAdminUser)
	routes := srv.GenerateRoutes()

	g := errgroup.Group{}
	respCh := make(chan *httptest.ResponseRecorder)
	g.Go(func() error {
		urlPath := "/api/grants?destination=deferred&lastUpdateIndex=1"
		req := httptest.NewRequest(http.MethodGet, urlPath, nil)
		req.Header.Set("Authorization", "Bearer "+adminAccessKey(srv))
		req.Header.Add("Infra-Version", apiVersionLatest)

		resp := httptest.NewRecorder()
		routes.ServeHTTP(resp, req)
		respCh <- resp
		return nil
	})

	isBlocked(t, respCh)

	// unrelated grant
	err := data.CreateGrant(srv.db, &models.Grant{
		Subject:   "i:abcd",
		Privilege: "view",
		Resource:  "somethingelse",
	})
	assert.NilError(t, err)
	isBlocked(t, respCh)

	// matching grant
	err = data.CreateGrant(srv.db, &models.Grant{
		Subject:   "i:abcd",
		Privilege: "view",
		Resource:  "deferred.ns1",
	})
	assert.NilError(t, err)

	resp := isNotBlocked(t, respCh)
	assert.Equal(t, resp.Code, http.StatusOK, (*responseDebug)(resp))
	assert.Equal(t, resp.Result().Header.Get("Last-Update-Index"), "10003", (*responseDebug)(resp))

	respBody := &api.ListResponse[api.Grant]{}
	assert.NilError(t, json.NewDecoder(resp.Body).Decode(respBody))

	assert.Equal(t, len(respBody.Items), 1)
}

func TestAPI_CreateGrant(t *testing.T) {
	srv := setupServer(t, withAdminUser, withMultiOrgEnabled)
	routes := srv.GenerateRoutes()

	keyID, _, _ := strings.Cut(adminAccessKey(srv), ".")
	accessKey, err := data.GetAccessKeyByKeyID(srv.DB(), keyID)
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

	otherOrg := createOtherOrg(t, srv.db)

	type testCase struct {
		setup    func(t *testing.T, req *http.Request)
		expected func(t *testing.T, resp *httptest.ResponseRecorder)
		body     api.GrantRequest
	}

	run := func(t *testing.T, tc testCase) {
		body := jsonBody(t, tc.body)
		req := httptest.NewRequest(http.MethodPost, "/api/grants", body)
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
			body: api.GrantRequest{},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusBadRequest, resp.Body.String())

				respBody := &api.Error{}
				err := json.Unmarshal(resp.Body.Bytes(), respBody)
				assert.NilError(t, err)

				expected := []api.FieldError{
					{Errors: []string{"one of (user, name, group) is required"}},
					{FieldName: "privilege", Errors: []string{"is required"}},
					{FieldName: "resource", Errors: []string{"is required"}},
				}
				assert.DeepEqual(t, respBody.FieldErrors, expected)
			},
		},
		"admin for wrong domain": {
			setup: func(t *testing.T, req *http.Request) {
				req.Host = "example.com"
				req.Header.Set("Authorization", "Bearer "+otherOrg.AdminAccessKey)
			},
			body: api.GrantRequest{
				User:      someUser.ID,
				Privilege: models.InfraAdminRole,
				Resource:  "some-cluster",
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusBadRequest, resp.Body.String())
			},
		},
		"success": {
			setup: func(t *testing.T, req *http.Request) {
				req.Header.Set("Authorization", "Bearer "+adminAccessKey(srv))
			},
			body: api.GrantRequest{
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
			body: api.GrantRequest{
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
			body: api.GrantRequest{
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
	srv := setupServer(t, withAdminUser, withMultiOrgEnabled)
	routes := srv.GenerateRoutes()

	user := &models.Identity{Name: "non-admin"}

	err := data.CreateIdentity(srv.DB(), user)
	assert.NilError(t, err)

	otherOrg := createOtherOrg(t, srv.db)

	t.Run("last infra admin is deleted", func(t *testing.T) {
		infraAdminGrants, err := data.ListGrants(srv.DB(), data.ListGrantsOptions{
			ByPrivileges: []string{models.InfraAdminRole},
			ByResource:   "infra",
		})
		assert.NilError(t, err)
		assert.Assert(t, len(infraAdminGrants) == 1)

		req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/grants/%s", infraAdminGrants[0].ID), nil)
		req.Header.Set("Authorization", "Bearer "+adminAccessKey(srv))
		req.Header.Set("Infra-Version", apiVersionLatest)

		resp := httptest.NewRecorder()
		routes.ServeHTTP(resp, req)

		assert.Equal(t, resp.Code, http.StatusBadRequest, resp.Body.String())
	})
	t.Run("admin for wrong organization", func(t *testing.T) {
		grant := &models.Grant{
			Subject:   uid.NewIdentityPolymorphicID(user.ID),
			Privilege: models.InfraViewRole,
			Resource:  "something",
		}
		err := data.CreateGrant(srv.DB(), grant)
		assert.NilError(t, err)

		req := httptest.NewRequest(http.MethodDelete, "/api/grants/"+grant.ID.String(), nil)
		req.Header.Set("Authorization", "Bearer "+otherOrg.AdminAccessKey)
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

func TestAPI_UpdateGrants(t *testing.T) {
	srv := setupServer(t, withAdminUser, withMultiOrgEnabled)
	routes := srv.GenerateRoutes()

	user := &models.Identity{Name: "non-admin"}

	err := data.CreateIdentity(srv.DB(), user)
	assert.NilError(t, err)

	type testCase struct {
		setup    func(t *testing.T, req *http.Request)
		expected func(t *testing.T, resp *httptest.ResponseRecorder)
		body     api.UpdateGrantsRequest
	}

	run := func(t *testing.T, tc testCase) {
		body := jsonBody(t, tc.body)
		req := httptest.NewRequest(http.MethodPatch, "/api/grants", body)
		req.Header.Add("Infra-Version", "0.15.2")

		if tc.setup != nil {
			tc.setup(t, req)
		}

		resp := httptest.NewRecorder()
		routes.ServeHTTP(resp, req)

		tc.expected(t, resp)
	}

	testCases := map[string]testCase{
		"success add": {
			setup: func(t *testing.T, req *http.Request) {
				req.Header.Set("Authorization", "Bearer "+adminAccessKey(srv))
			},
			body: api.UpdateGrantsRequest{
				GrantsToAdd: []api.GrantRequest{
					{
						User:      user.ID,
						Privilege: models.InfraAdminRole,
						Resource:  "some-cluster",
					},
				},
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusOK)

				infraAdminGrants, err := data.ListGrants(srv.DB(), data.ListGrantsOptions{
					ByPrivileges: []string{models.InfraAdminRole},
					ByResource:   "some-cluster",
				})
				assert.NilError(t, err)
				assert.Assert(t, len(infraAdminGrants) == 1)
			},
		},
		"success add w/ own username": {
			setup: func(t *testing.T, req *http.Request) {
				req.Header.Set("Authorization", "Bearer "+adminAccessKey(srv))
			},
			body: api.UpdateGrantsRequest{
				GrantsToAdd: []api.GrantRequest{
					{
						Name:      "admin@example.com",
						Privilege: models.InfraAdminRole,
						Resource:  "some-cluster2",
					},
				},
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusOK)

				infraAdminGrants, err := data.ListGrants(srv.DB(), data.ListGrantsOptions{
					ByPrivileges: []string{models.InfraAdminRole},
					ByResource:   "some-cluster2",
				})
				assert.NilError(t, err)
				assert.Assert(t, len(infraAdminGrants) == 1)
			},
		},
		"success add w/ username": {
			setup: func(t *testing.T, req *http.Request) {
				req.Header.Set("Authorization", "Bearer "+adminAccessKey(srv))
			},
			body: api.UpdateGrantsRequest{
				GrantsToAdd: []api.GrantRequest{
					{
						Name:      user.Name,
						Privilege: models.InfraAdminRole,
						Resource:  "some-cluster3",
					},
				},
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusOK)

				infraAdminGrants, err := data.ListGrants(srv.DB(), data.ListGrantsOptions{
					ByPrivileges: []string{models.InfraAdminRole},
					ByResource:   "some-cluster3",
				})
				assert.NilError(t, err)
				assert.Assert(t, len(infraAdminGrants) == 1)
			},
		},
		"success delete": {
			setup: func(t *testing.T, req *http.Request) {
				req.Header.Set("Authorization", "Bearer "+adminAccessKey(srv))

				grantToAdd := models.Grant{
					Subject:   uid.NewIdentityPolymorphicID(user.ID),
					Privilege: models.InfraAdminRole,
					Resource:  "another-cluster",
				}

				err := data.CreateGrant(srv.DB(), &grantToAdd)
				assert.NilError(t, err)
			},
			body: api.UpdateGrantsRequest{
				GrantsToRemove: []api.GrantRequest{
					{
						User:      user.ID,
						Privilege: models.InfraAdminRole,
						Resource:  "another-cluster",
					},
				},
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusOK)

				infraAdminGrants, err := data.ListGrants(srv.DB(), data.ListGrantsOptions{
					ByPrivileges: []string{models.InfraAdminRole},
					ByResource:   "another-cluster",
				})
				assert.NilError(t, err)
				assert.Assert(t, len(infraAdminGrants) == 0)
			},
		},
		"success delete w/ username": {
			setup: func(t *testing.T, req *http.Request) {
				req.Header.Set("Authorization", "Bearer "+adminAccessKey(srv))

				grantToAdd := models.Grant{
					Subject:   uid.NewIdentityPolymorphicID(user.ID),
					Privilege: models.InfraAdminRole,
					Resource:  "another-cluster2",
				}

				err := data.CreateGrant(srv.DB(), &grantToAdd)
				assert.NilError(t, err)
			},
			body: api.UpdateGrantsRequest{
				GrantsToRemove: []api.GrantRequest{
					{
						Name:      user.Name,
						Privilege: models.InfraAdminRole,
						Resource:  "another-cluster2",
					},
				},
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusOK)

				infraAdminGrants, err := data.ListGrants(srv.DB(), data.ListGrantsOptions{
					ByPrivileges: []string{models.InfraAdminRole},
					ByResource:   "another-cluster",
				})
				assert.NilError(t, err)
				assert.Assert(t, len(infraAdminGrants) == 0)
			},
		},
		"failure missing subject": {
			setup: func(t *testing.T, req *http.Request) {
				req.Header.Set("Authorization", "Bearer "+adminAccessKey(srv))
			},
			body: api.UpdateGrantsRequest{
				GrantsToAdd: []api.GrantRequest{
					{
						Privilege: models.InfraAdminRole,
						Resource:  "some-cluster",
					},
				},
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusBadRequest)
			},
		},
		"failure missing resource": {
			setup: func(t *testing.T, req *http.Request) {
				req.Header.Set("Authorization", "Bearer "+adminAccessKey(srv))
			},
			body: api.UpdateGrantsRequest{
				GrantsToAdd: []api.GrantRequest{
					{
						Name:      user.Name,
						Privilege: models.InfraAdminRole,
					},
				},
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusBadRequest)
			},
		},
		"failure missing privilege": {
			setup: func(t *testing.T, req *http.Request) {
				req.Header.Set("Authorization", "Bearer "+adminAccessKey(srv))
			},
			body: api.UpdateGrantsRequest{
				GrantsToAdd: []api.GrantRequest{
					{
						Name:     user.Name,
						Resource: "some-cluster",
					},
				},
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusBadRequest)
			},
		},
		"failure add w/ username": {
			setup: func(t *testing.T, req *http.Request) {
				req.Header.Set("Authorization", "Bearer "+adminAccessKey(srv))
			},
			body: api.UpdateGrantsRequest{
				GrantsToAdd: []api.GrantRequest{
					{
						Name:      "unknown@example.com",
						Privilege: models.InfraAdminRole,
						Resource:  "some-cluster",
					},
				},
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusBadRequest)
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			run(t, tc)
		})
	}

}
