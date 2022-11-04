package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"gotest.tools/v3/assert"

	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
)

func TestVerifyAndRedirect_Works(t *testing.T) {
	s := setupServer(t)

	routes := s.GenerateRoutes()

	user := &models.Identity{
		Name: "luna.lovegood@example.com",
	}
	createIdentities(t, s.db, user)
	assert.Assert(t, len(user.VerificationToken) > 0, "verification token must be set")
	assert.Assert(t, !user.Verified)

	redirectURL := "https://example.com/hello"
	url := wrapLinkWithVerification(redirectURL, "example.com", user.VerificationToken)

	req := httptest.NewRequest(http.MethodGet, url, nil)
	resp := httptest.NewRecorder()
	routes.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusPermanentRedirect, resp.Code, resp.Body.String())
	loc := resp.Result().Header.Get("Location")
	assert.Equal(t, loc, redirectURL)

	storedUser, err := data.GetIdentity(s.db, data.GetIdentityOptions{ByID: user.ID})
	assert.NilError(t, err)
	assert.Equal(t, storedUser.Verified, true)
}
