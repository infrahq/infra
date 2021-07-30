package registry

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"

	"go.uber.org/zap"
	"gopkg.in/square/go-jose.v2"
	"gorm.io/gorm"
)

var (
	CookieTokenName = "token"
	CookieLoginName = "login"
)

type Http struct {
	db     *gorm.DB
	logger *zap.Logger
}

func (h *Http) Healthz(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func (h *Http) WellKnownJWKs(w http.ResponseWriter, r *http.Request) {
	var settings Settings
	err := h.db.First(&settings).Error
	if err != nil {
		http.Error(w, "could not get JWKs", http.StatusInternalServerError)
		return
	}

	var pubKey jose.JSONWebKey
	err = pubKey.UnmarshalJSON(settings.PublicJWK)
	if err != nil {
		http.Error(w, "could not get JWKs", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(struct {
		Keys []jose.JSONWebKey `json:"keys"`
	}{
		[]jose.JSONWebKey{pubKey},
	})
}

func (h *Http) loginRedirectMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ext := filepath.Ext(r.URL.Path)
		if ext != "" && ext != ".html" {
			next.ServeHTTP(w, r)
			return
		}

		if strings.HasPrefix(r.URL.Path, "/_next") {
			next.ServeHTTP(w, r)
			return
		}

		token, tokenCookieErr := r.Cookie(CookieTokenName)
		if tokenCookieErr != nil && !errors.Is(tokenCookieErr, http.ErrNoCookie) {
			h.logger.Error(tokenCookieErr.Error())
			return
		}

		login, loginCookieErr := r.Cookie(CookieLoginName)
		if loginCookieErr != nil && !errors.Is(loginCookieErr, http.ErrNoCookie) {
			h.logger.Error(loginCookieErr.Error())
			return
		}

		// If the login or token cookie are missing, then redirect to /login or /signup based on the current status
		if errors.Is(loginCookieErr, http.ErrNoCookie) || errors.Is(tokenCookieErr, http.ErrNoCookie) {
			deleteAuthCookie(w)

			adminExists := h.db.Where(&User{Admin: true}).Find(&[]User{}).RowsAffected > 0
			if !adminExists && !strings.HasPrefix(r.URL.Path, "/signup") {
				http.Redirect(w, r, "/signup", http.StatusTemporaryRedirect)
				return
			} else if adminExists && !strings.HasPrefix(r.URL.Path, "/login") {
				params := url.Values{}
				path := "/login"

				next := ""
				if r.URL.Path != "/" {
					next += r.URL.Path
				}
				if r.URL.RawQuery != "" {
					next += "?" + r.URL.RawQuery
				}

				if next != "" {
					params.Add("next", next)
					path = "/login?" + params.Encode()
				}

				http.Redirect(w, r, path, http.StatusTemporaryRedirect)
				return
			}
		}

		// If the cookies exist, then validate their values and redirect to / or follow any ?next= query parameter
		if token != nil && login != nil {
			_, err := ValidateAndGetToken(h.db, token.Value)
			if err != nil {
				deleteAuthCookie(w)
				http.Redirect(w, r, "/login", http.StatusTemporaryRedirect)
			}

			if login.Value != "1" {
				deleteAuthCookie(w)
				http.Redirect(w, r, "/login", http.StatusTemporaryRedirect)
			}

			if strings.HasPrefix(r.URL.Path, "/login") || strings.HasPrefix(r.URL.Path, "/signup") {
				keys, ok := r.URL.Query()["next"]
				if !ok || len(keys[0]) < 1 {
					http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
					return
				} else {
					http.Redirect(w, r, keys[0], http.StatusTemporaryRedirect)
					return
				}
			}
		}

		next.ServeHTTP(w, r)
	})
}
