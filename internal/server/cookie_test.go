package server

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"gotest.tools/v3/assert"

	"github.com/infrahq/infra/internal/access"
	"github.com/infrahq/infra/internal/server/models"
)

func TestResendAuthCookie(t *testing.T) {
	baseDomain := "example.com"
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	req := &http.Request{
		Host:   fmt.Sprintf("dev.%s", baseDomain),
		Header: make(http.Header),
	}
	maxAge := int(time.Until(time.Now().Add(5 * time.Second)).Seconds())
	req.AddCookie(&http.Cookie{
		Name:     cookieSignupName,
		Value:    "aaa",
		MaxAge:   maxAge,
		Path:     cookiePath,
		Domain:   baseDomain,
		SameSite: http.SameSiteStrictMode,
		Secure:   true,
		HttpOnly: true,
	})
	c.Request = req
	rCtx := access.RequestContext{
		Request: c.Request,
		Authenticated: access.Authenticated{
			AccessKey: &models.AccessKey{
				ExpiresAt: time.Now().Add(5 * time.Minute),
			},
		},
	}
	c.Set(access.RequestContextKey, rCtx)

	bearer := exchangeSignupCookieForSession(c, Options{SessionDuration: 1 * time.Minute, BaseDomain: baseDomain})
	assert.Equal(t, "aaa", bearer)

	assert.Equal(t, len(c.Writer.Header()["Set-Cookie"]), 2)

	matched, err := regexp.MatchString("auth=aaa; Path=/; Domain=dev.example.com; Max-Age=\\d\\d; HttpOnly; SameSite=Strict", c.Writer.Header()["Set-Cookie"][0])
	assert.NilError(t, err)
	assert.Assert(t, matched)

	assert.Equal(t, "signup=; Path=/; Domain=example.com; Max-Age=0; HttpOnly; Secure", c.Writer.Header()["Set-Cookie"][1])
}
