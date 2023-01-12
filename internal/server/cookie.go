package server

import (
	"net/http"
	"net/url"
	"time"

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

func setCookie(req *http.Request, resp http.ResponseWriter, config cookieConfig) {
	maxAge := int(time.Until(config.Expires).Seconds())
	if maxAge == cookieMaxAgeNoExpiry {
		maxAge = cookieMaxAgeDeleteImmediately
	}

	secure := true
	if req.TLS == nil {
		// if the request came over HTTP, then the cookie will need to be sent unsecured
		secure = false
	}

	http.SetCookie(resp, &http.Cookie{
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

func deleteCookie(req *http.Request, resp http.ResponseWriter, name, domain string) {
	secure := true
	if req.TLS == nil {
		// if the request came over HTTP, then the cookie will need to be sent unsecured
		secure = false
	}

	http.SetCookie(resp, &http.Cookie{
		Name:     name,
		MaxAge:   cookieMaxAgeDeleteImmediately,
		Path:     cookiePath,
		Domain:   domain,
		Secure:   secure,
		HttpOnly: true, // not accessible by javascript
	})
}

// exchangeSignupCookieForSession sets the auth cookie on the current host making the request
func exchangeSignupCookieForSession(
	req *http.Request,
	resp http.ResponseWriter,
	opts Options,
) string {
	signupCookie, err := getCookie(req, cookieSignupName)
	if err != nil {
		logging.L.Trace().Err(err).Msg("failed to find signup cookie, this may be expected")
		return ""
	}

	exp := time.Now().UTC().Add(opts.SessionDuration)

	conf := cookieConfig{
		Name:    cookieAuthorizationName,
		Value:   signupCookie,
		Domain:  req.Host,
		Expires: exp,
	}
	setCookie(req, resp, conf)
	deleteCookie(req, resp, cookieSignupName, opts.BaseDomain)

	return signupCookie
}
