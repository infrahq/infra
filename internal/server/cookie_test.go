package server

import (
	"fmt"
	"net/http"
	"net/http/httptest"
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
		DBTxn:   nil,
		Authenticated: access.Authenticated{
			AccessKey: &models.AccessKey{
				ExpiresAt: time.Now().Add(5 * time.Minute),
			},
		},
	}
	c.Set(access.RequestContextKey, rCtx)

	exchangeSignupCookieForSession(c, baseDomain)

	assert.Equal(t, len(c.Writer.Header()["Set-Cookie"]), 2)
	expectedCookies := []string{
		"auth=aaa; Path=/; Domain=dev.example.com; Max-Age=299; HttpOnly; SameSite=Strict",
		"auth=; Path=/; Domain=example.com; Max-Age=0; HttpOnly; Secure",
	}
	assert.DeepEqual(t, c.Writer.Header()["Set-Cookie"], expectedCookies)
}
