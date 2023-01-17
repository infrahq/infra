package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	gocmp "github.com/google/go-cmp/cmp"
	"golang.org/x/crypto/bcrypt"
	"gotest.tools/v3/assert"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/access"
	"github.com/infrahq/infra/internal/generate"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/email"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

func TestAPI_GetUser(t *testing.T) {
	srv := setupServer(t, withAdminUser)
	routes := srv.GenerateRoutes()

	createID := func(t *testing.T, name string) uid.ID {
		t.Helper()
		var buf bytes.Buffer
		body := api.CreateUserRequest{Name: name}
		err := json.NewEncoder(&buf).Encode(body)
		assert.NilError(t, err)

		// nolint:noctx
		req := httptest.NewRequest(http.MethodPost, "/api/users", &buf)
		req.Header.Set("Authorization", "Bearer "+adminAccessKey(srv))
		req.Header.Set("Infra-Version", apiVersionLatest)

		resp := httptest.NewRecorder()
		routes.ServeHTTP(resp, req)
		assert.Equal(t, resp.Code, http.StatusCreated, resp.Body.String())
		respObj := &api.CreateUserResponse{}
		err = json.Unmarshal(resp.Body.Bytes(), respObj)
		assert.NilError(t, err)
		return respObj.ID
	}
	idMe := createID(t, "mememe@example.com")
	idHal := createID(t, "HAL@example.com")

	token := &models.AccessKey{
		IssuedForUser: idMe,
		ProviderID:    data.InfraProvider(srv.DB()).ID,
		ExpiresAt:     time.Now().Add(10 * time.Second),
	}

	accessKeyMe, err := data.CreateAccessKey(srv.DB(), token)
	assert.NilError(t, err)

	pubKey := `ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQDPkW3mIACvMmXqbeGF/U2MY8jbQ5NT24tRL0cl+32vRMmIDGcEyLkWh98D9qJlwCIZ8vJahAI3sqYJRoIHkiaRTslWwAZWNnTJ3TzeKUn/g0xutASD4znmQhNk3OuKPyuDKRxvsOuBVzuKiNNeUWVf5v/4gPrmBffS19cPPlHG+TwHNzTvyvbLcZu+xE18x8eCM4uRam0wa4RfHrMtaqPb/kFGz7skXv0/JFCXKrc//dMKHbr/brjj7fKYFYbMG7k15LewfZ/fLqsbJsvuP8OTIE7195fKhL1Gln8AKOM1E0CLX9nxK7qx4MlrDgEJBbqikWb2kVKmpxwcA7UcoUbwKZb4/QrOUDy22aHnIErIl2is9IP8RfBdKgzmgT1QmVPcGHI4gBAPb279zw58nAVp58gzHvK/oTDlAD2zq87i/PeDSzdoVZe0zliKOXAVzLQGI+9vsZ+6URHBe6J+Tj+PxOD5sWduhepOa/UKF96+CeEg/oso4UHR83z5zR38idc=`
	userPubKey := addUserPublicKey(t, srv.DB(), idMe, pubKey)

	type testCase struct {
		urlPath  string
		setup    func(t *testing.T, req *http.Request)
		expected func(t *testing.T, resp *httptest.ResponseRecorder)
	}

	run := func(t *testing.T, tc testCase) {
		req := httptest.NewRequest(http.MethodGet, tc.urlPath, nil)
		req.Header.Set("Authorization", "Bearer "+adminAccessKey(srv))
		req.Header.Set("Infra-Version", apiVersionLatest)

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
				key, _ := createAccessKey(t, srv.DB(), "someonenew@example.com")

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
					IssuedForUser: idMe,
					ProviderID:    data.InfraProvider(srv.DB()).ID,
					ExpiresAt:     time.Now().Add(10 * time.Second),
				}

				key, err := data.CreateAccessKey(srv.DB(), token)
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
						"name": "mememe@example.com",
						"lastSeenAt": "%[2]v",
						"created": "%[2]v",
						"providerNames": ["infra"],
						"sshLoginName": "mememe",
						"updated": "%[2]v",
						"publicKeys": [
							{
								"id": "<any-valid-uid>",
								"created": "%[2]v",
								"expires": "%[4]v",
								"fingerprint": "SHA256:dwF3R8L454kABUAJc+ZdJeaV2xbcXVJfb81tuv/1KLo",
								"publicKey": "%[3]v",
								"keyType": "ssh-rsa",
								"name": ""
							}
						]
					}`,
					idMe.String(),
					time.Now().UTC().Format(time.RFC3339),
					strings.Fields(pubKey)[1],
					userPubKey.Expires.Time().UTC().Format(time.RFC3339),
				))
				actual := jsonUnmarshal(t, resp.Body.String())

				cmpAPIUserJSON := gocmp.Options{
					gocmp.FilterPath(pathMapKey(`created`, `updated`, `lastSeenAt`), cmpApproximateTime),
					gocmp.FilterPath(pathMapKey(`id`), cmpAnyValidUID),
				}
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

func addUserPublicKey(t *testing.T, db *data.DB, userID uid.ID, key string) *api.UserPublicKey {
	t.Helper()
	tx := txnForTestCase(t, db, db.DefaultOrg.ID)
	c, _ := gin.CreateTestContext(nil)
	rCtx := access.RequestContext{}
	rCtx.DBTxn = tx
	rCtx.Authenticated.User = &models.Identity{Model: models.Model{ID: userID}}
	c.Set(access.RequestContextKey, rCtx)

	resp, err := AddUserPublicKey(c, &api.AddUserPublicKeyRequest{PublicKey: key})
	assert.NilError(t, err)
	assert.NilError(t, tx.Commit())
	return resp
}

func TestAPI_ListUsers(t *testing.T) {
	srv := setupServer(t, withAdminUser)
	routes := srv.GenerateRoutes()

	// TODO: Convert the "humans" group and "AnotherUser" user to call the standard http endpoints
	//       when the new endpoint to add a user to a group exists
	humans := models.Group{Name: "humans"}
	createGroups(t, srv.DB(), &humans)
	anotherID := models.Identity{
		Name:   "AnotherUser@example.com",
		Groups: []models.Group{humans},
	}
	createIdentities(t, srv.DB(), &anotherID)

	createID := func(t *testing.T, name string) uid.ID {
		t.Helper()
		var buf bytes.Buffer
		body := api.CreateUserRequest{Name: name}
		err := json.NewEncoder(&buf).Encode(body)
		assert.NilError(t, err)

		req := httptest.NewRequest(http.MethodPost, "/api/users", &buf)
		req.Header.Add("Authorization", "Bearer "+adminAccessKey(srv))
		req.Header.Add("Infra-Version", apiVersionLatest)

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

	pubKey := `ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQDPkW3mIACvMmXqbeGF/U2MY8jbQ5NT24tRL0cl+32vRMmIDGcEyLkWh98D9qJlwCIZ8vJahAI3sqYJRoIHkiaRTslWwAZWNnTJ3TzeKUn/g0xutASD4znmQhNk3OuKPyuDKRxvsOuBVzuKiNNeUWVf5v/4gPrmBffS19cPPlHG+TwHNzTvyvbLcZu+xE18x8eCM4uRam0wa4RfHrMtaqPb/kFGz7skXv0/JFCXKrc//dMKHbr/brjj7fKYFYbMG7k15LewfZ/fLqsbJsvuP8OTIE7195fKhL1Gln8AKOM1E0CLX9nxK7qx4MlrDgEJBbqikWb2kVKmpxwcA7UcoUbwKZb4/QrOUDy22aHnIErIl2is9IP8RfBdKgzmgT1QmVPcGHI4gBAPb279zw58nAVp58gzHvK/oTDlAD2zq87i/PeDSzdoVZe0zliKOXAVzLQGI+9vsZ+6URHBe6J+Tj+PxOD5sWduhepOa/UKF96+CeEg/oso4UHR83z5zR38idc=`
	addUserPublicKey(t, srv.DB(), id1, pubKey)

	type testCase struct {
		urlPath  string
		setup    func(t *testing.T, req *http.Request)
		expected func(t *testing.T, resp *httptest.ResponseRecorder)
	}

	run := func(t *testing.T, tc testCase) {
		// nolint:noctx
		req := httptest.NewRequest(http.MethodGet, tc.urlPath, nil)
		req.Header.Add("Authorization", "Bearer "+adminAccessKey(srv))
		req.Header.Add("Infra-Version", apiVersionLatest)

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
				assert.Equal(t, resp.Body.String(), `{"page":1,"limit":100,"totalPages":0,"totalCount":0,"count":0,"items":[]}`)
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
					PaginationResponse: api.PaginationResponse{Page: 1, Limit: 100, TotalPages: 1, TotalCount: 1},
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
					PaginationResponse: api.PaginationResponse{Page: 1, Limit: 100, TotalPages: 1, TotalCount: 3},
				}
				assert.DeepEqual(t, actual, expected, cmpAPIUserShallow)
			},
		},
		"no filter": {
			urlPath: "/api/users?showSystem=true",
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				if runtime.GOOS == "darwin" {
					t.Skip("this test doesn't do the right thing on mac due to a different default postgres sort order collation")
				}
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
					PaginationResponse: api.PaginationResponse{Page: 1, Limit: 100, TotalPages: 1, TotalCount: 7},
				}
				assert.DeepEqual(t, actual, expected, cmpAPIUserShallow)
			},
		},
		"hide connector": {
			urlPath: "/api/users",
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				if runtime.GOOS == "darwin" {
					t.Skip("this test doesn't do the right thing on mac due to a different default postgres sort order collation")
				}
				assert.Equal(t, resp.Code, http.StatusOK)

				var actual api.ListResponse[api.User]
				err := json.NewDecoder(resp.Body).Decode(&actual)
				assert.NilError(t, err)
				expected := api.ListResponse[api.User]{
					Count: 6,
					Items: []api.User{
						{Name: "AnotherUser@example.com"},
						{Name: "HAL@example.com"},
						{Name: "admin@example.com"},
						{Name: "me@example.com"},
						{Name: "other-HAL@example.com"},
						{Name: "other@example.com"},
					},
					PaginationResponse: api.PaginationResponse{Page: 1, Limit: 100, TotalPages: 1, TotalCount: 6},
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
			urlPath: "/api/users?limit=2&page=2&showSystem=true",
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				if runtime.GOOS == "darwin" {
					t.Skip("this test doesn't do the right thing on mac due to a different default postgres sort order collation")
				}

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
					PaginationResponse: api.PaginationResponse{Page: 1, Limit: 100, TotalPages: 1, TotalCount: 1},
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
					{FieldName: "limit", Errors: []string{"value 1001 must be at most 1000"}},
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
					{FieldName: "page", Errors: []string{"value -1 must be at least 0"}},
				}
				assert.DeepEqual(t, respBody.FieldErrors, expected)
			},
		},
		// TODO: assert full JSON response
		"query by public key fingerprint": {
			urlPath: "/api/users?publicKeyFingerprint=" + url.QueryEscape("SHA256:dwF3R8L454kABUAJc+ZdJeaV2xbcXVJfb81tuv/1KLo"),
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusOK, resp.Body.String())

				var actual api.ListResponse[api.User]
				err := json.NewDecoder(resp.Body).Decode(&actual)
				assert.NilError(t, err)

				expected := api.ListResponse[api.User]{
					Count: 1,
					Items: []api.User{
						{Name: "me@example.com"},
					},
					PaginationResponse: api.PaginationResponse{Page: 1, Limit: 100, TotalPages: 1, TotalCount: 1},
				}
				assert.DeepEqual(t, actual, expected, cmpAPIUserShallow)
				assert.Equal(t, len(actual.Items[0].PublicKeys), 1, "%#v", actual)
				assert.Equal(t, actual.Items[0].PublicKeys[0].Fingerprint, "SHA256:dwF3R8L454kABUAJc+ZdJeaV2xbcXVJfb81tuv/1KLo")
			},
		},
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
	routes := srv.GenerateRoutes()

	existing := &models.Identity{Name: "existing@example.com"}
	err := data.CreateIdentity(srv.DB(), existing)
	assert.NilError(t, err)

	type testCase struct {
		body     api.CreateUserRequest
		setup    func(t *testing.T, req *http.Request)
		expected func(t *testing.T, resp *httptest.ResponseRecorder)
	}

	run := func(t *testing.T, tc testCase) {
		body := jsonBody(t, tc.body)
		// nolint:noctx
		req := httptest.NewRequest(http.MethodPost, "/api/users", body)
		req.Header.Set("Authorization", "Bearer "+adminAccessKey(srv))
		req.Header.Set("Infra-Version", apiVersionLatest)

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
				key, _ := createAccessKey(t, srv.DB(), "someonenew@example.com")
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

				expected := []api.FieldError{
					{FieldName: "name", Errors: []string{"is required"}},
				}
				assert.DeepEqual(t, apiError.FieldErrors, expected)
			},
		},
		"invalid name": {
			body: api.CreateUserRequest{Name: "not an email"},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusBadRequest, resp.Body.String())

				var apiError api.Error
				err := json.NewDecoder(resp.Body).Decode(&apiError)
				assert.NilError(t, err)

				expected := []api.FieldError{
					{FieldName: "name", Errors: []string{"invalid email address"}},
				}
				assert.DeepEqual(t, apiError.FieldErrors, expected)
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
	srv := &Server{db: setupDB(t)}
	db := txnForTestCase(t, srv.db, srv.db.DefaultOrg.ID)

	a := &API{server: srv}
	admin := createAdmin(t, db)

	loginAs := func(tx *data.Transaction, user *models.Identity) *gin.Context {
		ctx, _ := gin.CreateTestContext(nil)
		ctx.Set(access.RequestContextKey, access.RequestContext{
			DBTxn:         tx,
			Authenticated: access.Authenticated{User: user},
		})
		return ctx
	}

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
				cred, err := data.GetCredentialByUserID(db, user.ID)
				assert.NilError(t, err)
				assert.Equal(t, true, cred.OneTimePassword)
			})
		})
		t.Run("as a user", func(t *testing.T) {
			ctx := loginAs(db, user)
			t.Run("with no existing infra user", func(t *testing.T) {
				err = data.DeleteProviderUsers(db, data.DeleteProviderUsersOptions{ByIdentityID: user.ID, ByProviderID: data.InfraProvider(db).ID})
				assert.NilError(t, err)

				t.Run("I cannot set a password", func(t *testing.T) {
					cred, err := data.GetCredentialByUserID(db, user.ID)
					assert.NilError(t, err)
					if cred != nil {
						_ = data.DeleteCredential(db, cred.ID)
					}

					_, err = a.UpdateUser(ctx, &api.UpdateUserRequest{
						ID:          user.ID,
						OldPassword: "whatever",
						Password:    "1234567890987654321a!",
					})
					assert.Error(t, err, "get credential: record not found")
				})
			})
			t.Run("with an existing infra user", func(t *testing.T) {
				_, _ = data.CreateProviderUser(db, data.InfraProvider(db), user)

				cred, _ := data.GetCredentialByUserID(db, user.ID)
				if cred != nil {
					_ = data.DeleteCredential(db, cred.ID)
				}

				hash, err := bcrypt.GenerateFromPassword([]byte("random password"), bcrypt.DefaultCost)
				assert.NilError(t, err)

				_ = data.CreateCredential(db, &models.Credential{
					IdentityID:   user.ID,
					PasswordHash: hash,
				})

				t.Run("I can change my password", func(t *testing.T) {
					_, err := a.UpdateUser(ctx, &api.UpdateUserRequest{
						ID:          user.ID,
						OldPassword: "random password",
						Password:    "1234567890987654321a!",
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

			hash, err := bcrypt.GenerateFromPassword([]byte("random password"), bcrypt.DefaultCost)
			assert.NilError(t, err)

			err = data.CreateCredential(db, &models.Credential{
				IdentityID:   user.ID,
				PasswordHash: hash,
			})
			assert.NilError(t, err)

			ctx := loginAs(db, user)
			t.Run("I can change my password", func(t *testing.T) {
				_, err := a.UpdateUser(ctx, &api.UpdateUserRequest{
					ID:          user.ID,
					OldPassword: "random password",
					Password:    "123454676twefdhsds",
				})
				assert.NilError(t, err)
			})
		})
	})
}

func TestAPI_CreateUser_EmailInvite(t *testing.T) {
	patchEmailTestMode(t, "fakekey")

	assert.Assert(t, email.IsConfigured())

	s := setupServer(t, withAdminUser)
	routes := s.GenerateRoutes()

	var token string
	runStep(t, "request user invite", func(t *testing.T) {
		body := jsonBody(t, &api.CreateUserRequest{Name: "deckard@example.com"})
		r := httptest.NewRequest(http.MethodPost, "/api/users", body)
		r.Header.Add("Infra-Version", apiVersionLatest)
		r.Header.Add("Authorization", fmt.Sprintf("Bearer %s", adminAccessKey(s)))

		w := httptest.NewRecorder()
		routes.ServeHTTP(w, r)
		assert.Equal(t, w.Code, http.StatusCreated, w.Body.String())
		assert.Equal(t, len(email.TestData), 1)

		data, ok := email.TestData[0].(email.UserInviteData)
		assert.Assert(t, ok)

		assert.Equal(t, data.FromUserName, "Admin")

		u, err := url.Parse(data.Link)
		assert.NilError(t, err)
		assert.Equal(t, u.Path, "/accept-invite")

		token = u.Query().Get("token")
		assert.Assert(t, token != "")
	})

	user, err := data.GetIdentity(s.DB(), data.GetIdentityOptions{ByName: "deckard@example.com"})
	assert.NilError(t, err)

	_, err = data.GetCredentialByUserID(s.DB(), user.ID)
	assert.ErrorIs(t, err, internal.ErrNotFound)

	_, err = data.GetProviderUser(s.DB(), data.InfraProvider(s.DB()).ID, user.ID)
	assert.ErrorIs(t, err, internal.ErrNotFound)

	// an invite is claimed by submitting a password reset request with the invite token
	runStep(t, "claim invite token", func(t *testing.T) {
		body := jsonBody(t, &api.VerifiedResetPasswordRequest{Token: token, Password: "mysecret"})
		r := httptest.NewRequest(http.MethodPost, "/api/password-reset", body)
		r.Header.Add("Infra-Version", apiVersionLatest)

		w := httptest.NewRecorder()
		routes.ServeHTTP(w, r)
		assert.Equal(t, w.Code, http.StatusCreated, w.Body.String())

		credential, err := data.GetCredentialByUserID(s.DB(), user.ID)
		assert.NilError(t, err)

		err = bcrypt.CompareHashAndPassword(credential.PasswordHash, []byte("mysecret"))
		assert.NilError(t, err)

		_, err = data.GetProviderUser(s.DB(), data.InfraProvider(s.DB()).ID, user.ID)
		assert.NilError(t, err)
	})
}

func TestAPI_DeleteUser(t *testing.T) {
	srv := setupServer(t, withAdminUser)
	routes := srv.GenerateRoutes()

	connector := data.InfraConnectorIdentity(srv.DB())

	selfKey, selfUser := createAccessKey(t, srv.DB(), "user@example.com")

	testUser := &models.Identity{Name: "test"}
	err := data.CreateIdentity(srv.DB(), testUser)
	assert.NilError(t, err)

	type testCase struct {
		name     string
		urlPath  string
		setup    func(t *testing.T, req *http.Request)
		expected func(t *testing.T, resp *httptest.ResponseRecorder)
	}

	run := func(t *testing.T, tc testCase) {
		// nolint:noctx
		req := httptest.NewRequest(http.MethodDelete, tc.urlPath, nil)
		req.Header.Set("Authorization", "Bearer "+adminAccessKey(srv))
		req.Header.Set("Infra-Version", apiVersionLatest)

		if tc.setup != nil {
			tc.setup(t, req)
		}
		resp := httptest.NewRecorder()
		routes.ServeHTTP(resp, req)

		tc.expected(t, resp)
	}

	testCases := []testCase{
		// TODO: not authenticated
		// TODO: not authorized
		{
			name:    "can not delete internal users",
			urlPath: "/api/users/" + connector.ID.String(),
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusBadRequest, resp.Code, resp.Body.String())
			},
		},
		{
			name:    "can not delete self",
			urlPath: "/api/users/" + selfUser.ID.String(),
			setup: func(t *testing.T, req *http.Request) {
				req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", selfKey))
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusBadRequest, resp.Code, resp.Body.String())
			},
		},
		{
			name:    "success",
			urlPath: "/api/users/" + testUser.ID.String(),
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusNoContent, resp.Code, resp.Body.String())
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			run(t, tc)
		})
	}
}

func TestAPI_UpdateUser(t *testing.T) {
	srv := setupServer(t, withAdminUser)
	routes := srv.GenerateRoutes()

	run := func(t *testing.T, request *api.UpdateUserRequest, id, accessKey string) *httptest.ResponseRecorder {
		body := jsonBody(t, request)

		r := httptest.NewRequest(http.MethodPut, "/api/users/"+id, body)
		r.Header.Set("Infra-Version", apiVersionLatest)

		if accessKey != "" {
			r.Header.Set("Authorization", "Bearer "+accessKey)
		}

		w := httptest.NewRecorder()
		routes.ServeHTTP(w, r)
		return w
	}

	t.Run("update own user", func(t *testing.T) {
		user := &models.Identity{Name: "salsa@example.com"}
		err := data.CreateIdentity(srv.DB(), user)
		assert.NilError(t, err)

		accessKey := &models.AccessKey{IssuedForUser: user.ID, ProviderID: data.InfraProvider(srv.DB()).ID}
		accessKeySecret, err := data.CreateAccessKey(srv.DB(), accessKey)
		assert.NilError(t, err)

		t.Run("cannot create their own credential", func(t *testing.T) {
			response := run(t, &api.UpdateUserRequest{}, user.ID.String(), accessKeySecret)
			assert.Equal(t, response.Code, http.StatusNotFound, response.Body.String())
		})

		hash, err := bcrypt.GenerateFromPassword([]byte("supersecret"), bcrypt.DefaultCost)
		assert.NilError(t, err)

		err = data.CreateCredential(srv.DB(), &models.Credential{IdentityID: user.ID, PasswordHash: hash})
		assert.NilError(t, err)

		t.Run("unauthenticated", func(t *testing.T) {
			request := &api.UpdateUserRequest{}
			response := run(t, request, user.ID.String(), "")
			assert.Equal(t, response.Code, http.StatusUnauthorized, response.Body.String())
		})

		t.Run("old password empty", func(t *testing.T) {
			request := &api.UpdateUserRequest{}
			response := run(t, request, user.ID.String(), accessKeySecret)
			assert.Equal(t, response.Code, http.StatusBadRequest, response.Body.String())

			var body api.Error
			err := json.Unmarshal(response.Body.Bytes(), &body)
			assert.NilError(t, err)

			expected := []api.FieldError{{FieldName: "oldPassword", Errors: []string{"invalid password"}}}
			assert.DeepEqual(t, body.FieldErrors, expected)
		})

		t.Run("old password mismatch", func(t *testing.T) {
			request := &api.UpdateUserRequest{OldPassword: "notsupersecret"}
			response := run(t, request, user.ID.String(), accessKeySecret)
			assert.Equal(t, response.Code, http.StatusBadRequest, response.Body.String())

			var body api.Error
			err := json.Unmarshal(response.Body.Bytes(), &body)
			assert.NilError(t, err)

			expected := []api.FieldError{{FieldName: "oldPassword", Errors: []string{"invalid password"}}}
			assert.DeepEqual(t, body.FieldErrors, expected)
		})

		t.Run("new password invalid", func(t *testing.T) {
			request := &api.UpdateUserRequest{OldPassword: "supersecret", Password: "short"}
			response := run(t, request, user.ID.String(), accessKeySecret)
			assert.Equal(t, response.Code, http.StatusBadRequest, response.Body.String())

			var body api.Error
			err := json.Unmarshal(response.Body.Bytes(), &body)
			assert.NilError(t, err)

			expected := []api.FieldError{{FieldName: "password", Errors: []string{"8 characters"}}}
			assert.DeepEqual(t, body.FieldErrors, expected)
		})

		t.Run("can change their own password", func(t *testing.T) {
			request := &api.UpdateUserRequest{OldPassword: "supersecret", Password: "mysecret"}
			response := run(t, request, user.ID.String(), accessKeySecret)
			assert.Equal(t, response.Code, http.StatusOK, response.Body.String())

			var body api.UpdateUserResponse
			err := json.Unmarshal(response.Body.Bytes(), &body)
			assert.NilError(t, err)

			assert.Equal(t, body.OneTimePassword, "")
		})

		t.Run("changing own password unsets password reset scope", func(t *testing.T) {
			accessKey := &models.AccessKey{
				IssuedForUser: user.ID,
				ProviderID:    data.InfraProvider(srv.DB()).ID,
				Scopes:        models.CommaSeparatedStrings{models.ScopePasswordReset},
			}

			accessKeySecret, err := data.CreateAccessKey(srv.DB(), accessKey)
			assert.NilError(t, err)

			request := &api.UpdateUserRequest{OldPassword: "mysecret", Password: "mysecret"}
			response := run(t, request, user.ID.String(), accessKeySecret)
			assert.Equal(t, response.Code, http.StatusOK, response.Body.String())

			accessKey, err = data.GetAccessKey(srv.DB(), data.GetAccessKeysOptions{ByID: accessKey.ID})
			assert.NilError(t, err)
			assert.DeepEqual(t, accessKey.Scopes, models.CommaSeparatedStrings{})
		})
	})

	t.Run("update other users", func(t *testing.T) {
		user := &models.Identity{Name: "tango@example.com"}
		err := data.CreateIdentity(srv.DB(), user)
		assert.NilError(t, err)

		t.Run("unauthenticated", func(t *testing.T) {
			request := &api.UpdateUserRequest{}
			response := run(t, request, user.ID.String(), "")
			assert.Equal(t, response.Code, http.StatusUnauthorized, response.Body.String())
		})

		t.Run("unauthorized", func(t *testing.T) {
			accessKey, _ := createAccessKey(t, srv.DB(), "notadmin@example.com")

			request := &api.UpdateUserRequest{}
			response := run(t, request, user.ID.String(), accessKey)
			assert.Equal(t, response.Code, http.StatusForbidden, response.Body.String())
		})

		t.Run("can create credential", func(t *testing.T) {
			request := &api.UpdateUserRequest{}
			response := run(t, request, user.ID.String(), adminAccessKey(srv))
			assert.Equal(t, response.Code, http.StatusOK, response.Body.String())

			var body api.UpdateUserResponse
			err := json.Unmarshal(response.Body.Bytes(), &body)
			assert.NilError(t, err)

			assert.Assert(t, body.OneTimePassword != "")

			credential, err := data.GetCredentialByUserID(srv.DB(), user.ID)
			assert.NilError(t, err)

			assert.Assert(t, credential.OneTimePassword)
		})

		t.Run("can reset credential to a specific value", func(t *testing.T) {
			request := &api.UpdateUserRequest{Password: "mysecret"}
			response := run(t, request, user.ID.String(), adminAccessKey(srv))
			assert.Equal(t, response.Code, http.StatusOK, response.Body.String())

			var body api.UpdateUserResponse
			err := json.Unmarshal(response.Body.Bytes(), &body)
			assert.NilError(t, err)

			assert.Equal(t, body.OneTimePassword, "mysecret")

			credential, err := data.GetCredentialByUserID(srv.DB(), user.ID)
			assert.NilError(t, err)

			err = bcrypt.CompareHashAndPassword(credential.PasswordHash, []byte(body.OneTimePassword))
			assert.NilError(t, err)
		})

		t.Run("can reset credential to a random value", func(t *testing.T) {
			request := &api.UpdateUserRequest{}
			response := run(t, request, user.ID.String(), adminAccessKey(srv))
			assert.Equal(t, response.Code, http.StatusOK, response.Body.String())

			var body api.UpdateUserResponse
			err := json.Unmarshal(response.Body.Bytes(), &body)
			assert.NilError(t, err)

			assert.Assert(t, body.OneTimePassword != "")

			credential, err := data.GetCredentialByUserID(srv.DB(), user.ID)
			assert.NilError(t, err)

			err = bcrypt.CompareHashAndPassword(credential.PasswordHash, []byte(body.OneTimePassword))
			assert.NilError(t, err)
		})
	})
}

func TestAddUserPublicKey(t *testing.T) {
	srv := setupServer(t, withAdminUser)
	routes := srv.GenerateRoutes()

	type testCase struct {
		name     string
		setup    func(t *testing.T, req *http.Request)
		body     func(t *testing.T) api.AddUserPublicKeyRequest
		expected func(t *testing.T, resp *httptest.ResponseRecorder)
	}

	run := func(t *testing.T, tc testCase) {
		body := tc.body(t)

		req := httptest.NewRequest(http.MethodPut, "/api/users/public-key", jsonBody(t, body))
		req.Header.Set("Authorization", "Bearer "+adminAccessKey(srv))
		req.Header.Set("Infra-Version", apiVersionLatest)

		if tc.setup != nil {
			tc.setup(t, req)
		}
		resp := httptest.NewRecorder()
		routes.ServeHTTP(resp, req)

		tc.expected(t, resp)
	}

	cmpAPIPublicKeyJSON := gocmp.Options{
		gocmp.FilterPath(pathMapKey(`created`), cmpApproximateTime),
		gocmp.FilterPath(pathMapKey(`id`), cmpAnyValidUID),
	}

	testCases := []testCase{
		{
			name: "missing authentication",
			setup: func(t *testing.T, req *http.Request) {
				req.Header.Del("Authorization")
			},
			body: func(t *testing.T) api.AddUserPublicKeyRequest {
				return api.AddUserPublicKeyRequest{}
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusUnauthorized, (*responseDebug)(resp))
			},
		},
		{
			name: "success",
			body: func(t *testing.T) api.AddUserPublicKeyRequest {
				pubKey := `ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQDPkW3mIACvMmXqbeGF/U2MY8jbQ5NT24tRL0cl+32vRMmIDGcEyLkWh98D9qJlwCIZ8vJahAI3sqYJRoIHkiaRTslWwAZWNnTJ3TzeKUn/g0xutASD4znmQhNk3OuKPyuDKRxvsOuBVzuKiNNeUWVf5v/4gPrmBffS19cPPlHG+TwHNzTvyvbLcZu+xE18x8eCM4uRam0wa4RfHrMtaqPb/kFGz7skXv0/JFCXKrc//dMKHbr/brjj7fKYFYbMG7k15LewfZ/fLqsbJsvuP8OTIE7195fKhL1Gln8AKOM1E0CLX9nxK7qx4MlrDgEJBbqikWb2kVKmpxwcA7UcoUbwKZb4/QrOUDy22aHnIErIl2is9IP8RfBdKgzmgT1QmVPcGHI4gBAPb279zw58nAVp58gzHvK/oTDlAD2zq87i/PeDSzdoVZe0zliKOXAVzLQGI+9vsZ+6URHBe6J+Tj+PxOD5sWduhepOa/UKF96+CeEg/oso4UHR83z5zR38idc=`
				return api.AddUserPublicKeyRequest{
					Name:      "the-name",
					PublicKey: pubKey,
				}
			},
			expected: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, resp.Code, http.StatusOK, (*responseDebug)(resp))

				actual := jsonUnmarshal(t, resp.Body.String())
				expected := jsonUnmarshal(t, fmt.Sprintf(`
{
	"id": "<any-valid-uid>",
	"created": "%[1]v",
	"expires": "%[2]v",
	"fingerprint": "SHA256:dwF3R8L454kABUAJc+ZdJeaV2xbcXVJfb81tuv/1KLo",
	"keyType": "ssh-rsa",
	"name": "the-name",
	"publicKey": "AAAAB3NzaC1yc2EAAAADAQABAAABgQDPkW3mIACvMmXqbeGF/U2MY8jbQ5NT24tRL0cl+32vRMmIDGcEyLkWh98D9qJlwCIZ8vJahAI3sqYJRoIHkiaRTslWwAZWNnTJ3TzeKUn/g0xutASD4znmQhNk3OuKPyuDKRxvsOuBVzuKiNNeUWVf5v/4gPrmBffS19cPPlHG+TwHNzTvyvbLcZu+xE18x8eCM4uRam0wa4RfHrMtaqPb/kFGz7skXv0/JFCXKrc//dMKHbr/brjj7fKYFYbMG7k15LewfZ/fLqsbJsvuP8OTIE7195fKhL1Gln8AKOM1E0CLX9nxK7qx4MlrDgEJBbqikWb2kVKmpxwcA7UcoUbwKZb4/QrOUDy22aHnIErIl2is9IP8RfBdKgzmgT1QmVPcGHI4gBAPb279zw58nAVp58gzHvK/oTDlAD2zq87i/PeDSzdoVZe0zliKOXAVzLQGI+9vsZ+6URHBe6J+Tj+PxOD5sWduhepOa/UKF96+CeEg/oso4UHR83z5zR38idc="
}`,
					time.Now().Format(time.RFC3339),
					time.Now().UTC().Add(12*time.Hour).Format(time.RFC3339)))

				assert.DeepEqual(t, actual, expected, cmpAPIPublicKeyJSON)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			run(t, tc)
		})
	}
}
