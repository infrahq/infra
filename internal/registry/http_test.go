package registry

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLoginRedirectMiddlewarePassthrough(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "favicon")
	}

	db, err := NewDB("file::memory:")
	if err != nil {
		t.Fatal(err)
	}

	httpHandlers := &Http{
		db: db,
	}

	r := httptest.NewRequest("GET", "http://test.com/favicon.ico", nil)
	w := httptest.NewRecorder()
	httpHandlers.loginRedirectMiddleware(http.HandlerFunc(handler)).ServeHTTP(w, r)
	assert.Equal(t, w.Code, http.StatusOK)
}

func TestLoginRedirectMiddlewareNext(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "hello world")
	}

	db, err := NewDB("file::memory:")
	if err != nil {
		t.Fatal(err)
	}

	httpHandlers := &Http{
		db: db,
	}

	r := httptest.NewRequest("GET", "http://test.com/_next/file", nil)
	w := httptest.NewRecorder()
	httpHandlers.loginRedirectMiddleware(http.HandlerFunc(handler)).ServeHTTP(w, r)
	assert.Equal(t, w.Code, http.StatusOK)
}

func TestLoginRedirectSetsNextParameter(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "hello world")
	}

	db, err := NewDB("file::memory:")
	if err != nil {
		t.Fatal(err)
	}

	httpHandlers := &Http{
		db: db,
	}

	r := httptest.NewRequest("GET", "http://test.com/dashboard?param=1", nil)
	w := httptest.NewRecorder()
	httpHandlers.loginRedirectMiddleware(http.HandlerFunc(handler)).ServeHTTP(w, r)
	assert.Equal(t, w.Code, http.StatusTemporaryRedirect)
	assert.Equal(t, w.Header().Get("Location"), "/login?next=%2Fdashboard%3Fparam%3D1")
}

func TestLoginRedirectNoRedirectIfLoggedIn(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "hello world")
	}

	db, err := NewDB("file::memory:")
	if err != nil {
		t.Fatal(err)
	}

	httpHandlers := &Http{
		db: db,
	}

	id, secret, err := addUser(db, time.Hour*24)
	if err != nil {
		t.Fatal(err)
	}

	r := httptest.NewRequest("GET", "http://test.com/dashboard", nil)
	expires := time.Now().Add(SessionDuration)
	r.AddCookie(&http.Cookie{
		Name:     CookieTokenName,
		Value:    id + secret,
		Expires:  expires,
		HttpOnly: true,
		Path:     "/",
		SameSite: http.SameSiteStrictMode,
	})

	r.AddCookie(&http.Cookie{
		Name:    CookieLoginName,
		Value:   "1",
		Expires: expires,
		Path:    "/",
	})

	w := httptest.NewRecorder()
	httpHandlers.loginRedirectMiddleware(http.HandlerFunc(handler)).ServeHTTP(w, r)
	assert.Equal(t, w.Code, http.StatusOK)
}

func TestLoginRedirectIfLoginCookieUnset(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "hello world")
	}

	db, err := NewDB("file::memory:")
	if err != nil {
		t.Fatal(err)
	}

	httpHandlers := &Http{
		db: db,
	}

	id, secret, err := addUser(db, time.Hour*24)
	if err != nil {
		t.Fatal(err)
	}

	r := httptest.NewRequest("GET", "http://test.com/dashboard", nil)
	expires := time.Now().Add(SessionDuration)
	r.AddCookie(&http.Cookie{
		Name:     CookieTokenName,
		Value:    id + secret,
		Expires:  expires,
		HttpOnly: true,
		Path:     "/",
		SameSite: http.SameSiteStrictMode,
	})

	w := httptest.NewRecorder()
	httpHandlers.loginRedirectMiddleware(http.HandlerFunc(handler)).ServeHTTP(w, r)
	assert.Equal(t, w.Code, http.StatusTemporaryRedirect)
	assert.Equal(t, w.Header().Get("Location"), "/login?next=%2Fdashboard")

	for _, cookie := range w.Result().Cookies() {
		if cookie.Name == CookieTokenName {
			assert.Equal(t, cookie.Value, "")
		}
	}
}

func TestLoginRedirectFromLoginIfAlreadyLoggedIn(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "hello world")
	}

	db, err := NewDB("file::memory:")
	if err != nil {
		t.Fatal(err)
	}

	httpHandlers := &Http{
		db: db,
	}

	id, secret, err := addUser(db, time.Hour*24)
	if err != nil {
		t.Fatal(err)
	}

	r := httptest.NewRequest("GET", "http://test.com/login?next=/dashboard", nil)
	expires := time.Now().Add(SessionDuration)
	r.AddCookie(&http.Cookie{
		Name:     CookieTokenName,
		Value:    id + secret,
		Expires:  expires,
		HttpOnly: true,
		Path:     "/",
		SameSite: http.SameSiteStrictMode,
	})

	r.AddCookie(&http.Cookie{
		Name:    CookieLoginName,
		Value:   "1",
		Expires: expires,
		Path:    "/",
	})

	w := httptest.NewRecorder()
	httpHandlers.loginRedirectMiddleware(http.HandlerFunc(handler)).ServeHTTP(w, r)
	assert.Equal(t, w.Code, http.StatusTemporaryRedirect)
	assert.Equal(t, w.Header().Get("Location"), "/dashboard")
}
