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
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

func TestAPI_ListGrants(t *testing.T) {
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
		req.Header.Add("Infra-Version", "0.12.3")

		resp := httptest.NewRecorder()
		routes.ServeHTTP(resp, req)
		assert.Equal(t, resp.Code, http.StatusCreated, resp.Body.String())
		respObj := &api.CreateIdentityResponse{}
		err = json.Unmarshal(resp.Body.Bytes(), respObj)
		assert.NilError(t, err)
		return respObj.ID
	}

	createGrant := func(t *testing.T, user uid.ID, privilege string) {
		t.Helper()
		var buf bytes.Buffer
		body := api.CreateGrantRequest{
			Identity:  user,
			Privilege: privilege,
			Resource:  "res1",
		}
		err := json.NewEncoder(&buf).Encode(body)
		assert.NilError(t, err)

		req, err := http.NewRequest(http.MethodPost, "/v1/grants", &buf)
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
		for _, user := range users {
			iden := models.Identity{Model: models.Model{ID: user}}
			group.Identities = append(group.Identities, iden)
		}

		err := data.CreateGroup(srv.db, group)
		assert.NilError(t, err)
		return group.ID
	}

	idInGroup := createID(t, "inagroup@example.com")
	idOther := createID(t, "other@example.com")

	createGrant(t, idInGroup, "custom1")
	createGrant(t, idOther, "custom2")

	groupID := createGroup(t, "humans", idInGroup)

	token := &models.AccessKey{
		IssuedFor:  idInGroup,
		ProviderID: data.InfraProvider(srv.db).ID,
		ExpiresAt:  time.Now().Add(10 * time.Second),
	}

	accessKey, err := data.CreateAccessKey(srv.db, token)
	assert.NilError(t, err)

	admin, err := data.GetIdentity(srv.db, data.ByName("admin"))
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
			urlPath: "/v1/grants?subject=i:" + idOther.String(),
			setup: func(t *testing.T, req *http.Request) {
				req.Header.Del("Authorization")
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusUnauthorized)
			},
		},
		"not authorized, wrong identity": {
			urlPath: "/v1/grants?subject=i:" + idOther.String(),
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusForbidden, resp.Body.String())
			},
		},
		"not authorized, wrong group": {
			urlPath: "/v1/grants?subject=g:abcde",
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusForbidden, resp.Body.String())
			},
		},
		"not authorized, no subject in query": {
			urlPath: "/v1/grants?resource=res1",
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusForbidden, resp.Body.String())
			},
		},
		"authorized by grant": {
			urlPath: "/v1/grants?resource=none",
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
			urlPath: "/v1/grants?identity=" + idInGroup.String(),
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusOK, resp.Body.String())
				var grants api.ListResponse[api.Grant]
				err := json.NewDecoder(resp.Body).Decode(&grants)
				assert.NilError(t, err)
				expected := []api.Grant{
					{
						Identity:  idInGroup,
						Privilege: "custom1",
						Resource:  "res1",
					},
				}
				assert.DeepEqual(t, grants.Items, expected, cmpAPIGrantShallow)
			},
		},
		"authorized by group matching subject": {
			urlPath: "/v1/grants?group=" + groupID.String(),
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusOK, resp.Body.String())
				var grants api.ListResponse[api.Grant]
				err := json.NewDecoder(resp.Body).Decode(&grants)
				assert.NilError(t, err)
				// no grants for this group
				assert.Equal(t, len(grants.Items), 0)
			},
		},
		"no filters": {
			urlPath: "/v1/grants",
			setup: func(t *testing.T, req *http.Request) {
				req.Header.Set("Authorization", "Bearer "+adminAccessKey(srv))
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				connector, err := data.GetIdentity(srv.db, data.ByName("connector"))
				assert.NilError(t, err)

				assert.Equal(t, resp.Code, http.StatusOK, resp.Body.String())
				var grants api.ListResponse[api.Grant]
				err = json.NewDecoder(resp.Body).Decode(&grants)
				assert.NilError(t, err)

				expected := []api.Grant{
					{
						Identity:  admin.ID,
						Privilege: "admin",
						Resource:  "infra",
					},
					{
						Identity:  connector.ID,
						Privilege: "connector",
						Resource:  "infra",
					},
					{
						Identity:  idInGroup,
						Privilege: "custom1",
						Resource:  "res1",
					},
					{
						Identity:  idOther,
						Privilege: "custom2",
						Resource:  "res1",
					},
				}
				assert.DeepEqual(t, grants.Items, expected, cmpAPIGrantShallow)
			},
		},
		"filter by resource": {
			urlPath: "/v1/grants?resource=res1",
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
						Identity:  idInGroup,
						Privilege: "custom1",
						Resource:  "res1",
					},
					{
						Identity:  idOther,
						Privilege: "custom2",
						Resource:  "res1",
					},
				}
				assert.DeepEqual(t, grants.Items, expected, cmpAPIGrantShallow)
			},
		},
		"filter by resource and privilege": {
			urlPath: "/v1/grants?resource=res1&privilege=custom1",
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
						Identity:  idInGroup,
						Privilege: "custom1",
						Resource:  "res1",
					},
				}
				assert.DeepEqual(t, grants.Items, expected, cmpAPIGrantShallow)
			},
		},
		"full JSON response": {
			urlPath: "/v1/grants?identity=" + idInGroup.String(),
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusOK, resp.Body.String())

				expected := jsonUnmarshal(t, fmt.Sprintf(`
{
	"count": 1,
	"items": [{
		"id": "<any-valid-uid>",
		"created_by": "%[1]v",
		"privilege": "custom1",
		"resource": "res1",
		"identity": "%[2]v",
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

var cmpAPIGrantShallow = gocmp.Comparer(func(x, y api.Grant) bool {
	return x.Identity == y.Identity && x.Privilege == y.Privilege && x.Resource == y.Resource
})

var cmpAPIGrantJSON = gocmp.Options{
	gocmp.FilterPath(pathMapKey(`created`, `updated`), cmpApproximateTime),
	gocmp.FilterPath(pathMapKey(`id`), cmpAnyValidUID),
}
