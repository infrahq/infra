package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"runtime"
	"testing"

	"gotest.tools/v3/assert"

	"github.com/infrahq/infra/internal"
)

func TestErrorStatusCode(t *testing.T) {
	codes := []int{
		http.StatusContinue,
		http.StatusOK,
		http.StatusCreated,
		http.StatusMovedPermanently,
		http.StatusBadRequest,
		http.StatusUnauthorized,
		http.StatusForbidden,
		http.StatusNotFound,
		http.StatusInternalServerError,
		http.StatusNotImplemented,
		http.StatusBadGateway,
	}

	t.Run("equal to self, not equal to other codes", func(t *testing.T) {
		for c := 0; c < len(codes); c++ {
			err := Error{Code: int32(codes[c])}
			for o := 0; o < len(codes); o++ {
				if o == c {
					assert.Equal(t, ErrorStatusCode(err), int32(codes[o]))
					continue
				}

				assert.Assert(t, ErrorStatusCode(err) != int32(codes[o]),
					"code=%v, other=%v", err.Code, codes[o])
			}
		}
	})

	t.Run("nil error returns 0", func(t *testing.T) {
		assert.Equal(t, ErrorStatusCode(nil), int32(0))
	})

	t.Run("other errors return 0", func(t *testing.T) {
		assert.Equal(t, ErrorStatusCode(fmt.Errorf("other error")), int32(0))
	})

	t.Run("from wrapped error", func(t *testing.T) {
		err := fmt.Errorf("with some wrapping: %w",
			Error{Code: int32(http.StatusInternalServerError)})

		actual := ErrorStatusCode(err)
		assert.Equal(t, actual, int32(http.StatusInternalServerError))
	})
}

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
			_ = json.NewEncoder(resp).Encode(Error{
				Code:    http.StatusBadRequest,
				Message: "bad request: it failed because",
			})
		default:
			resp.WriteHeader(http.StatusInternalServerError)
		}
	}
	srv := httptest.NewServer(http.HandlerFunc(handler))

	c := Client{
		Name:      "testing",
		Version:   "version",
		URL:       srv.URL,
		AccessKey: "the-access-key",
	}

	type stubResponse struct{}

	expectedHeaders := http.Header{
		"Accept-Encoding": []string{"gzip"},
		"Authorization":   []string{"Bearer the-access-key"},
		"Content-Type":    []string{"application/json"},
		"Accept":          []string{"application/json"},
		"Infra-Version":   []string{internal.FullVersion()},
		"User-Agent":      []string{fmt.Sprintf("Infra/%v (testing version; %v/%v)", internal.FullVersion(), runtime.GOOS, runtime.GOARCH)},
	}

	ctx := context.Background()

	t.Run("success request", func(t *testing.T) {
		_, err := get[stubResponse](ctx, c, "/good", Query{})
		assert.NilError(t, err)
		req := <-requestCh
		assert.Equal(t, req.Method, http.MethodGet)
		assert.Equal(t, req.URL.Path, "/good")
		assert.DeepEqual(t, req.Header, expectedHeaders)
	})

	t.Run("bad request", func(t *testing.T) {
		_, err := get[stubResponse](ctx, c, "/bad", Query{})
		assert.Error(t, err, `bad request: it failed because`)
		req := <-requestCh
		assert.Equal(t, req.Method, http.MethodGet)
		assert.Equal(t, req.URL.Path, "/bad")
		assert.DeepEqual(t, req.Header, expectedHeaders)
	})

	t.Run("server error", func(t *testing.T) {
		_, err := get[stubResponse](ctx, c, "/invalid", Query{})
		assert.Error(t, err, `500 internal server error`)
		req := <-requestCh
		assert.Equal(t, req.Method, http.MethodGet)
		assert.Equal(t, req.URL.Path, "/invalid")
		assert.DeepEqual(t, req.Header, expectedHeaders)
	})
}

func TestDelete(t *testing.T) {
	ctx := context.Background()
	ch := make(chan *http.Request, 1)
	handler := func(rw http.ResponseWriter, r *http.Request) {
		ch <- r
		rw.WriteHeader(http.StatusOK)
	}

	srv := httptest.NewServer(http.HandlerFunc(handler))

	c := Client{
		Name:      "testing",
		Version:   "version",
		AccessKey: "access-key",
		URL:       srv.URL,
	}

	expected := http.Header{
		"Accept-Encoding": {"gzip"},
		"Authorization":   {"Bearer access-key"},
		"Content-Type":    {"application/json"},
		"Accept":          {"application/json"},
		"Infra-Version":   {internal.FullVersion()},
		"User-Agent":      {fmt.Sprintf("Infra/%v (testing version; %v/%v)", internal.FullVersion(), runtime.GOOS, runtime.GOARCH)},
	}

	t.Run("headers", func(t *testing.T) {
		err := delete(ctx, c, "/good", Query{})
		assert.NilError(t, err)

		r := <-ch
		assert.DeepEqual(t, r.Header, expected)
		assert.Equal(t, r.Method, http.MethodDelete)
		assert.Equal(t, r.URL.Path, "/good")
	})
}

func TestListGrants(t *testing.T) {
	reqCh := make(chan *http.Request, 1)
	handler := func(resp http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			resp.WriteHeader(http.StatusInternalServerError)
			return
		}

		switch r.URL.Path {
		case "/api/grants":
			reqCh <- r

			lastUpdateIndex := r.URL.Query().Get("lastUpdateIndex")
			if lastUpdateIndex == "70000" {
				resp.WriteHeader(http.StatusNotModified)
				return
			}

			resp.Header().Set("Last-Update-Index", "10010")
			resp.WriteHeader(http.StatusOK)
			_, _ = resp.Write([]byte(`{}`))
		default:
			resp.WriteHeader(http.StatusInternalServerError)
		}
	}
	srv := httptest.NewServer(http.HandlerFunc(handler))

	c := Client{
		Name:      "testing",
		Version:   "version",
		URL:       srv.URL,
		AccessKey: "the-access-key",
	}

	ctx := context.Background()

	t.Run("sets value from Last-Update-Index header", func(t *testing.T) {
		resp, err := c.ListGrants(ctx, ListGrantsRequest{
			Resource:        "anything",
			BlockingRequest: BlockingRequest{LastUpdateIndex: 1234},
		})
		assert.NilError(t, err)

		assert.Equal(t, resp.LastUpdateIndex.Index, int64(10010))

		req := <-reqCh
		assert.Equal(t, req.URL.Query().Get("lastUpdateIndex"), "1234")
	})
	t.Run("not modified", func(t *testing.T) {
		_, err := c.ListGrants(ctx, ListGrantsRequest{
			Resource:        "anything",
			BlockingRequest: BlockingRequest{LastUpdateIndex: 70000},
		})
		var apiError Error
		assert.Assert(t, errors.As(err, &apiError), err)
		expected := Error{Code: http.StatusNotModified}
		assert.DeepEqual(t, apiError, expected)
	})
}

func TestVersionAhead(t *testing.T) {
	t.Run("patch version", func(t *testing.T) {
		res, err := versionAhead("1.0.1", "1.0.0")
		assert.NilError(t, err)
		assert.Assert(t, res)

		res, err = versionAhead("1.0.0", "1.0.1")
		assert.NilError(t, err)
		assert.Assert(t, !res)
	})
	t.Run("minor version", func(t *testing.T) {
		res, err := versionAhead("1.1.0", "1.0.0")
		assert.NilError(t, err)
		assert.Assert(t, res)

		res, err = versionAhead("1.0.0", "1.1.0")
		assert.NilError(t, err)
		assert.Assert(t, !res)
	})
	t.Run("major version", func(t *testing.T) {
		res, err := versionAhead("2.0.0", "1.0.0")
		assert.NilError(t, err)
		assert.Assert(t, res)

		res, err = versionAhead("1.0.0", "2.0.0")
		assert.NilError(t, err)
		assert.Assert(t, !res)
	})
	t.Run("equal", func(t *testing.T) {
		res, err := versionAhead("1.0.0", "1.0.0")
		assert.NilError(t, err)
		assert.Assert(t, !res)
	})
}
