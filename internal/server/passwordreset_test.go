package server

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"gotest.tools/v3/assert"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/email"
	"github.com/infrahq/infra/internal/server/models"
)

func TestPasswordResetFlow(t *testing.T) {
	s := setupServer(t)
	routes := s.GenerateRoutes()

	email.TestMode = true

	user := &models.Identity{
		Name: "skeletor@example.com",
	}

	err := data.CreateIdentity(s.DB(), user)
	assert.NilError(t, err)

	err = data.CreateCredential(s.DB(), &models.Credential{
		IdentityID:   user.ID,
		PasswordHash: []byte("foo"),
	})
	assert.NilError(t, err)

	// request password reset
	body, err := json.Marshal(&api.PasswordResetRequest{
		Email: "skeletor@example.com",
	})
	assert.NilError(t, err)

	// nolint:noctx
	req := httptest.NewRequest(http.MethodPost, "/api/password-reset-request", bytes.NewBuffer(body))
	req.Header.Add("Infra-Version", "0.13.6")

	resp := httptest.NewRecorder()
	routes.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusCreated, resp.Code, resp.Body.String())

	assert.Assert(t, len(email.TestDataSent) > 0)

	// cheat and grab the token from the db.
	tokens := []string{}
	err = s.db.Raw("select token from password_reset_tokens").Pluck("token", &tokens).Error
	assert.NilError(t, err)

	assert.Assert(t, len(tokens) > 0)
	token := tokens[len(tokens)-1]

	resetData, ok := email.TestDataSent[0].(email.PasswordResetData)
	assert.Assert(t, ok)
	// TODO: fix test so that we can verify the domain; default org has blank domain
	u, err := url.Parse(resetData.Link)
	assert.NilError(t, err)
	link, err := base64.URLEncoding.DecodeString(u.Query().Get("r"))
	assert.NilError(t, err)
	assert.Equal(t, string(link), "https:///password-reset?token="+token)

	// reset the password with the token
	body, err = json.Marshal(&api.VerifiedResetPasswordRequest{
		Token:    token,
		Password: "my new pw!2351",
	})
	assert.NilError(t, err)

	// nolint:noctx
	req = httptest.NewRequest(http.MethodPost, "/api/password-reset", bytes.NewBuffer(body))
	req.Header.Add("Infra-Version", "0.13.6")

	resp = httptest.NewRecorder()
	routes.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusCreated, resp.Code, resp.Body.String())

}
