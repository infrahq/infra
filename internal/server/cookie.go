package server

import (
	"net/http"
	"net/url"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/infrahq/infra/internal/logging"
)

const (
	cookieAuthorizationName       = "auth"
	cookieSignupName              = "signup"
	cookiePath                    = "/"
	cookieMaxAgeDeleteImmediately = -1 // <0: delete immediately
	cookieMaxAgeNoExpiry          = 0  // zero has special meaning of "no expiry"
)

type cookieConfig struct {
	Name    string
	Value   string
	Domain  string
	Expires time.Time
}

func setCookie(c *gin.Context, config cookieConfig) {
	maxAge := int(time.Until(config.Expires).Seconds())
	if maxAge == cookieMaxAgeNoExpiry {
		maxAge = cookieMaxAgeDeleteImmediately
	}

	secure := true
	if c.Request.TLS == nil {
		// if the request came over HTTP, then the cookie will need to be sent unsecured
		secure = false
	}

	http.SetCookie(c.Writer, &http.Cookie{
		Name:     config.Name,
		Value:    url.QueryEscape(config.Value),
		MaxAge:   maxAge,
		Path:     cookiePath,
		Domain:   config.Domain,
		SameSite: http.SameSiteStrictMode,
		Secure:   secure,
		HttpOnly: true, // not accessible by javascript
	})
}

func deleteCookie(c *gin.Context, name, domain string) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     cookieAuthorizationName,
		MaxAge:   cookieMaxAgeDeleteImmediately,
		Path:     cookiePath,
		Domain:   domain,
		Secure:   true, // only over https
		HttpOnly: true, // not accessible by javascript
	})
}

// exchangeSignupCookieForSession sets the auth cookie on the current host making the request
func exchangeSignupCookieForSession(c *gin.Context, baseDomain string) {
	key := getRequestContext(c).Authenticated.AccessKey // this should have been set by the middleware
	if key != nil {
		signupCookie, err := getCookie(c.Request, cookieSignupName)
		if err != nil {
			logging.L.Trace().Err(err).Msg("failed to find signup cookie, this may be expected")
			return
		}
		conf := cookieConfig{
			Name:    cookieAuthorizationName,
			Value:   signupCookie,
			Domain:  c.Request.Host,
			Expires: key.ExpiresAt,
		}
		setCookie(c, conf)
		deleteCookie(c, cookieSignupName, baseDomain)
	}
}
