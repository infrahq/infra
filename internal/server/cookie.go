package server

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

var (
	CookieAuthorizationName = "auth"
	CookieLoginName         = "login"
	CookieDomain            = ""
	CookiePath              = "/"
	// while these vars look goofy, they avoid "magic number" arguments to SetCookie
	CookieHTTPOnlyNotJavascriptAccessible = true    // setting HttpOnly to true means JS can't access it.
	CookieSecureHTTPSOnly                 = true    // setting Secure to true means the cookie is only sent over https connections
	CookieMaxAgeDeleteImmediately         = int(-1) // <0: delete immediately
	CookieMaxAgeNoExpiry                  = int(0)  // zero has special meaning of "no expiry"
)

func setAuthCookie(c *gin.Context, key string, sessionDuration time.Duration) {
	expires := time.Now().Add(sessionDuration)

	maxAge := int(time.Until(expires).Seconds())
	if maxAge == CookieMaxAgeNoExpiry {
		maxAge = CookieMaxAgeDeleteImmediately
	}

	c.SetSameSite(http.SameSiteStrictMode)

	c.SetCookie(CookieAuthorizationName, key, maxAge, CookiePath, CookieDomain, CookieSecureHTTPSOnly, CookieHTTPOnlyNotJavascriptAccessible)
	c.SetCookie(CookieLoginName, "1", maxAge, CookiePath, CookieDomain, CookieSecureHTTPSOnly, CookieHTTPOnlyNotJavascriptAccessible)
}

func deleteAuthCookie(c *gin.Context) {
	c.SetCookie(CookieAuthorizationName, "", CookieMaxAgeDeleteImmediately, CookiePath, CookieDomain, CookieSecureHTTPSOnly, CookieHTTPOnlyNotJavascriptAccessible)
	c.SetCookie(CookieLoginName, "", CookieMaxAgeDeleteImmediately, CookiePath, CookieDomain, CookieSecureHTTPSOnly, CookieHTTPOnlyNotJavascriptAccessible)
}
