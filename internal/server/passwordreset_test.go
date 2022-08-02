package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"gotest.tools/v3/assert"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/email"
	"github.com/infrahq/infra/internal/server/models"
)

func TestPasswordResetFlow(t *testing.T) {
	s := setupServer(t)
	routes := s.GenerateRoutes(prometheus.NewRegistry())

	email.TestMode = true

	user := &models.Identity{
		Name: "skeletor@example.com",
	}

	err := data.CreateIdentity(s.db, user)
	assert.NilError(t, err)

	err = data.CreateCredential(s.db, &models.Credential{
		IdentityID:   user.ID,
		PasswordHash: []byte("foo"),
	})
	assert.NilError(t, err)

	// request password reset
	body, err := json.Marshal(&api.PasswordResetRequest{
		Email: "skeletor@example.com",
	})
	assert.NilError(t, err)

	req, err := http.NewRequest(http.MethodPost, "/api/password-reset-request", bytes.NewBuffer(body))
	assert.NilError(t, err)

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

	assert.Equal(t, email.TestDataSent[0]["link"], "https://infrahq.com/password-reset?token="+token)

	// reset the password with the token
	body, err = json.Marshal(&api.VerifiedResetPasswordRequest{
		Token:    token,
		Password: "my new pw!2351",
	})
	assert.NilError(t, err)

	req, err = http.NewRequest(http.MethodPost, "/api/password-reset", bytes.NewBuffer(body))
	assert.NilError(t, err)

	req.Header.Add("Infra-Version", "0.13.6")

	resp = httptest.NewRecorder()
	routes.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusCreated, resp.Code, resp.Body.String())

}
