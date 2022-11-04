package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"gotest.tools/v3/assert"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/uid"
)

func TestServerLimitsAccessWithTemporaryPassword(t *testing.T) {
	srv := setupServer(t, withAdminUser)
	routes := srv.GenerateRoutes()

	// create a user
	resp := createUser(t, srv, routes, "hubert@example.com")

	// user can login with temporary password
	loginResp := login(t, routes, "hubert@example.com", resp.OneTimePassword)

	key := loginResp.AccessKey

	// user can't access other urls.
	tryOtherURL := func() *httptest.ResponseRecorder {
		// nolint:noctx
		req := httptest.NewRequest(http.MethodGet, "/api/users/"+loginResp.UserID.String(), nil)
		req.Header.Add("Authorization", "Bearer "+key)
		req.Header.Add("Infra-Version", "0.14")

		resp1 := httptest.NewRecorder()
		routes.ServeHTTP(resp1, req)
		return resp1
	}

	resp1 := tryOtherURL()
	assert.Equal(t, http.StatusUnauthorized, resp1.Code)

	// change password
	changePassword(t, routes, key, loginResp.UserID, resp.OneTimePassword, "balloons")

	// can access other urls.
	resp2 := tryOtherURL()

	assert.Equal(t, http.StatusOK, resp2.Code)
}

func changePassword(t *testing.T, routes Routes, accessKey string, id uid.ID, oldPassword, password string) *api.User {
	r := &api.UpdateUserRequest{
		OldPassword: oldPassword,
		Password:    password,
	}
	body, err := json.Marshal(r)
	assert.NilError(t, err)

	// nolint:noctx
	req := httptest.NewRequest(http.MethodPut, "/api/users/"+id.String(), bytes.NewReader(body))
	req.Header.Add("Authorization", "Bearer "+accessKey)
	req.Header.Add("Infra-Version", "0.14")

	resp := httptest.NewRecorder()
	routes.ServeHTTP(resp, req)

	assert.Equal(t, 200, resp.Code)

	result := &api.User{}
	err = json.Unmarshal(resp.Body.Bytes(), result)
	assert.NilError(t, err)

	return result
}

// login does an http login and returns the access key
func login(t *testing.T, routes Routes, name, pass string) *api.LoginResponse {
	loginReq := api.LoginRequest{PasswordCredentials: &api.LoginRequestPasswordCredentials{Name: name, Password: pass}}
	body := jsonBody(t, loginReq)
	req := httptest.NewRequest(http.MethodPost, "/api/login", body)
	req.Header.Add("Infra-Version", "0.14.0")

	resp := httptest.NewRecorder()
	routes.ServeHTTP(resp, req)

	assert.Equal(t, 201, resp.Code)

	loginResp := &api.LoginResponse{}

	err := json.Unmarshal(resp.Body.Bytes(), loginResp)
	assert.NilError(t, err)

	return loginResp
}
