package server

import (
	"net/http"
	"net/url"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	cookieAuthorizationName       = "auth"
	cookieLoginName               = "login"
	cookiePath                    = "/"
	cookieMaxAgeDeleteImmediately = -1 // <0: delete immediately
	cookieMaxAgeNoExpiry          = 0  // zero has special meaning of "no expiry"
)

func setAuthCookie(c *gin.Context, key string, expires time.Time) {
	maxAge := int(time.Until(expires).Seconds())
	if maxAge == cookieMaxAgeNoExpiry {
		maxAge = cookieMaxAgeDeleteImmediately
	}

	secure := true
	if c.Request.TLS == nil {
		// if the request came over HTTP, then the cookie will need to be sent unsecured
		secure = false
	}

	http.SetCookie(c.Writer, &http.Cookie{
		Name:     cookieAuthorizationName,
		Value:    url.QueryEscape(key),
		MaxAge:   maxAge,
		Path:     cookiePath,
		Domain:   c.Request.Host,
		SameSite: http.SameSiteStrictMode,
		Secure:   secure,
		HttpOnly: true, // not accessible by javascript
	})
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     cookieLoginName,
		Value:    "1",
		MaxAge:   maxAge,
		Path:     cookiePath,
		Domain:   c.Request.Host,
		SameSite: http.SameSiteStrictMode,
		Secure:   secure,
		HttpOnly: true, // not accessible by javascript
	})
}

func deleteAuthCookie(c *gin.Context) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     cookieAuthorizationName,
		MaxAge:   cookieMaxAgeDeleteImmediately,
		Path:     cookiePath,
		Domain:   c.Request.Host,
		Secure:   true, // only over https
		HttpOnly: true, // not accessible by javascript
	})

	http.SetCookie(c.Writer, &http.Cookie{
		Name:     cookieLoginName,
		MaxAge:   cookieMaxAgeDeleteImmediately,
		Path:     cookiePath,
		Domain:   c.Request.Host,
		Secure:   true, // only over https
		HttpOnly: true, // not accessible by javascript
	})
}
