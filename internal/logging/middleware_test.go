package logging

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"gotest.tools/v3/assert"
)

func TestMiddleware(t *testing.T) {
	setup := func(t *testing.T, writer io.Writer) *gin.Engine {
		PatchLogger(t, writer)

		router := gin.New()
		router.Use(Middleware())

		router.GET("/good", func(c *gin.Context) {
			assert.Equal(t, c.Request.Method, http.MethodGet)
			assert.Equal(t, c.Request.URL.Path, "/good")
		})

		router.GET("/gooder", func(c *gin.Context) {
			assert.Equal(t, c.Request.Method, http.MethodGet)
			assert.Equal(t, c.Request.URL.Path, "/gooder")
		})

		router.GET("/bad", func(c *gin.Context) {
			assert.Equal(t, c.Request.Method, http.MethodGet)
			assert.Equal(t, c.Request.URL.Path, "/bad")
			_ = c.AbortWithError(http.StatusBadRequest, errors.New("something went wrong"))
		})

		return router
	}

	t.Run("good", func(t *testing.T) {
		b := &bytes.Buffer{}
		router := setup(t, b)
		router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/good", nil))
		router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/good", nil))

		lines := bytes.Split(bytes.TrimSpace(b.Bytes()), []byte("\n"))
		assert.Equal(t, len(lines), 1)

		good := map[string]interface{}{}
		err := json.Unmarshal(lines[0], &good)
		assert.NilError(t, err)
		assert.Equal(t, good["path"], "/good")
	})

	t.Run("good and gooder", func(t *testing.T) {
		b := &bytes.Buffer{}
		router := setup(t, b)
		router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/good", nil))
		router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/gooder", nil))

		lines := bytes.Split(bytes.TrimSpace(b.Bytes()), []byte("\n"))
		assert.Equal(t, len(lines), 2)

		good := map[string]interface{}{}
		err := json.Unmarshal(lines[0], &good)
		assert.NilError(t, err)
		assert.Equal(t, good["path"], "/good")

		gooder := map[string]interface{}{}
		err = json.Unmarshal(lines[1], &gooder)
		assert.NilError(t, err)
		assert.Equal(t, gooder["path"], "/gooder")
	})

	t.Run("bad bad bad", func(t *testing.T) {
		b := &bytes.Buffer{}
		router := setup(t, b)
		router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/bad", nil))
		router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/bad", nil))
		router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/bad", nil))

		lines := bytes.Split(bytes.TrimSpace(b.Bytes()), []byte("\n"))
		assert.Equal(t, len(lines), 3)

		for _, line := range lines {
			bad := map[string]interface{}{}
			err := json.Unmarshal(line, &bad)
			assert.NilError(t, err)
			assert.Equal(t, bad["path"], "/bad")
		}
	})
}
