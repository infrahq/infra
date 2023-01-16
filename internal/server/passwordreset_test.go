package server

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"golang.org/x/crypto/bcrypt"
	"gotest.tools/v3/assert"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/email"
	"github.com/infrahq/infra/internal/server/models"
)

func patchEmailTestMode(t *testing.T) {
	t.Helper()
	email.TestMode = true
	t.Cleanup(func() {
		email.TestMode = false
		email.TestData = make([]any, 0)
	})
}

func TestPasswordResetFlow(t *testing.T) {
	patchEmailTestMode(t)

	s := setupServer(t)
	routes := s.GenerateRoutes()

	user := &models.Identity{Name: "skeletor@example.com"}
	err := data.CreateIdentity(s.DB(), user)
	assert.NilError(t, err)

	credential := &models.Credential{IdentityID: user.ID, PasswordHash: []byte("password")}
	err = data.CreateCredential(s.DB(), credential)
	assert.NilError(t, err)

	var token string
	runStep(t, "request password reset", func(t *testing.T) {
		body := jsonBody(t, &api.PasswordResetRequest{Email: "skeletor@example.com"})
		r := httptest.NewRequest(http.MethodPost, "/api/password-reset-request", body)
		r.Header.Add("Infra-Version", apiVersionLatest)

		w := httptest.NewRecorder()
		routes.ServeHTTP(w, r)
		assert.Equal(t, w.Code, http.StatusCreated, w.Body.String())
		assert.Equal(t, len(email.TestData), 1)

		data, ok := email.TestData[0].(email.PasswordResetData)
		assert.Assert(t, ok)

		u, err := url.Parse(data.Link)
		assert.NilError(t, err)
		assert.Equal(t, u.Path, "/link")

		link, err := base64.URLEncoding.DecodeString(u.Query().Get("r"))
		assert.NilError(t, err)

		u, err = url.Parse(string(link))
		assert.NilError(t, err)
		assert.Equal(t, u.Path, "/password-reset")

		token = u.Query().Get("token")
		assert.Assert(t, token != "")
	})

	runStep(t, "new password empty", func(t *testing.T) {
		body := jsonBody(t, &api.VerifiedResetPasswordRequest{Token: token})
		r := httptest.NewRequest(http.MethodPost, "/api/password-reset", body)
		r.Header.Add("Infra-Version", apiVersionLatest)

		w := httptest.NewRecorder()
		routes.ServeHTTP(w, r)
		assert.Equal(t, w.Code, http.StatusBadRequest, w.Body.String())
	})

	runStep(t, "new password does not satisfy password policy", func(t *testing.T) {
		body := jsonBody(t, &api.VerifiedResetPasswordRequest{Token: token, Password: "secret"})
		r := httptest.NewRequest(http.MethodPost, "/api/password-reset", body)
		r.Header.Add("Infra-Version", apiVersionLatest)

		w := httptest.NewRecorder()
		routes.ServeHTTP(w, r)
		assert.Equal(t, w.Code, http.StatusBadRequest, w.Body.String())
	})

	runStep(t, "claim password reset token", func(t *testing.T) {
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
	})

	runStep(t, "password reset token cannot be claimed more than once", func(t *testing.T) {
		body := jsonBody(t, &api.VerifiedResetPasswordRequest{Token: token, Password: "mysecret"})
		r := httptest.NewRequest(http.MethodPost, "/api/password-reset", body)
		r.Header.Add("Infra-Version", apiVersionLatest)

		w := httptest.NewRecorder()
		routes.ServeHTTP(w, r)
		assert.Equal(t, w.Code, http.StatusNotFound, w.Body.String())
	})
}
