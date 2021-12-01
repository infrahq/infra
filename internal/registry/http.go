package registry

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gopkg.in/square/go-jose.v2"
	"gorm.io/gorm"

	"github.com/infrahq/infra/internal/registry/data"
)

var (
	CookieTokenName = "token"
	CookieLoginName = "login"
	CookieDomain    = ""
	CookiePath      = "/"
	// while these vars look goofy, they avoid "magic number" arguments to SetCookie
	CookieHTTPOnlyJavascriptAccessible    = false   // setting HttpOnly to false means JS can access it.
	CookieHTTPOnlyNotJavascriptAccessible = true    // setting HttpOnly to true means JS can't access it.
	CookieSecureHTTPSOnly                 = true    // setting Secure to true means the cookie is only sent over https connections
	CookieSecureHttpOrHTTPS               = false   // setting Secure to false means the cookie will be sent over http or https connections
	CookieMaxAgeDeleteImmediately         = int(-1) // <0: delete immediately
	CookieMaxAgeNoExpiry                  = int(0)  // zero has special meaning of "no expiry"
)

func setAuthCookie(c *gin.Context, token string, sessionDuration time.Duration) {
	expires := time.Now().Add(sessionDuration)

	maxAge := int(time.Until(expires).Seconds())
	if maxAge == CookieMaxAgeNoExpiry {
		maxAge = CookieMaxAgeDeleteImmediately
	}

	c.SetSameSite(http.SameSiteStrictMode)

	c.SetCookie(CookieTokenName, token, maxAge, CookiePath, CookieDomain, CookieSecureHTTPSOnly, CookieHTTPOnlyNotJavascriptAccessible)
	c.SetCookie(CookieLoginName, "1", maxAge, CookiePath, CookieDomain, CookieSecureHTTPSOnly, CookieHTTPOnlyNotJavascriptAccessible)
}

func deleteAuthCookie(c *gin.Context) {
	c.SetCookie(CookieTokenName, "", CookieMaxAgeDeleteImmediately, CookiePath, CookieDomain, CookieSecureHTTPSOnly, CookieHTTPOnlyNotJavascriptAccessible)
	c.SetCookie(CookieLoginName, "", CookieMaxAgeDeleteImmediately, CookiePath, CookieDomain, CookieSecureHTTPSOnly, CookieHTTPOnlyNotJavascriptAccessible)
}

type HTTP struct {
	db *gorm.DB
}

func Healthz(c *gin.Context) {
	c.Status(http.StatusOK)
}

func (h *HTTP) WellKnownJWKs(c *gin.Context) {
	settings, err := data.GetSettings(h.db)
	if err != nil {
		sendAPIError(c, http.StatusInternalServerError, fmt.Errorf("could not get JWKs"))
		return
	}

	var pubKey jose.JSONWebKey
	if err := pubKey.UnmarshalJSON(settings.PublicJWK); err != nil {
		sendAPIError(c, http.StatusInternalServerError, fmt.Errorf("could not get JWKs"))
		return
	}

	c.JSON(http.StatusOK, struct {
		Keys []jose.JSONWebKey `json:"keys"`
	}{
		[]jose.JSONWebKey{pubKey},
	})
}
