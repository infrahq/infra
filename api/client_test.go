package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"gotest.tools/v3/assert"
)

func TestGet(t *testing.T) {
	requestCh := make(chan *http.Request, 5)
	handler := func(resp http.ResponseWriter, r *http.Request) {
		requestCh <- r
		switch r.URL.Path {
		case "/good":
			resp.WriteHeader(http.StatusOK)
			_, _ = resp.Write([]byte(`{}`))
		case "/bad":
			resp.WriteHeader(http.StatusBadRequest)
		default:
			resp.WriteHeader(http.StatusInternalServerError)
		}
	}
	srv := httptest.NewServer(http.HandlerFunc(handler))

	c := Client{
		URL:       srv.URL,
		AccessKey: "the-access-key",
		Headers:   http.Header{},
	}
	c.Headers.Add("User-Agent", "testing")
	c.Headers.Add("X-Custom", "custom")

	type stubResponse struct{}

	expectedHeaders := http.Header{
		"User-Agent":      []string{"testing"},
		"X-Custom":        []string{"custom"},
		"Accept-Encoding": []string{"gzip"},
		"Authorization":   []string{"Bearer the-access-key"},
	}

	t.Run("success request", func(t *testing.T) {
		_, err := get[stubResponse](c, "/good")
		assert.NilError(t, err)
		req := <-requestCh
		assert.Equal(t, req.Method, http.MethodGet)
		assert.Equal(t, req.URL.Path, "/good")
		assert.DeepEqual(t, req.Header, expectedHeaders)
	})

	t.Run("failed request", func(t *testing.T) {
		_, err := get[stubResponse](c, "/bad")
		assert.ErrorContains(t, err, `responded 400`)
		req := <-requestCh
		assert.Equal(t, req.Method, http.MethodGet)
		assert.Equal(t, req.URL.Path, "/bad")
		assert.DeepEqual(t, req.Header, expectedHeaders)
	})
}
