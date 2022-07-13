package logging

import (
	"bytes"
	"encoding/json"
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

		router.GET("/good", func(c *gin.Context) {})
		router.POST("/good", func(c *gin.Context) {})
		router.GET("/gooder", func(c *gin.Context) {})
		router.GET("/bad", func(c *gin.Context) {
			c.Status(http.StatusBadRequest)
		})
		router.GET("/broken", func(c *gin.Context) {
			c.Status(http.StatusBadGateway)
		})

		return router
	}

	t.Run("identical requests are sampled", func(t *testing.T) {
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

	t.Run("non-200 status responses are never sampled", func(t *testing.T) {
		b := &bytes.Buffer{}
		router := setup(t, b)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, httptest.NewRequest("GET", "/bad", nil))
		router.ServeHTTP(resp, httptest.NewRequest("GET", "/bad", nil))
		router.ServeHTTP(resp, httptest.NewRequest("GET", "/bad", nil))

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
