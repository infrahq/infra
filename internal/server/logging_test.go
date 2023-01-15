package server

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

	"github.com/infrahq/infra/internal/access"
	"github.com/infrahq/infra/internal/logging"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

func TestLoggingMiddleware(t *testing.T) {
	setup := func(t *testing.T, writer io.Writer) *gin.Engine {
		logging.PatchLogger(t, writer)

		router := gin.New()
		router.Use(loggingMiddleware(true))

		router.GET("/good/:id", func(c *gin.Context) {})
		router.POST("/good/:id", func(c *gin.Context) {})
		router.GET("/gooder/", func(c *gin.Context) {})
		router.GET("/bad/:id", func(c *gin.Context) {
			c.Status(http.StatusBadRequest)
		})
		router.GET("/broken", func(c *gin.Context) {
			c.Status(http.StatusInternalServerError)
		})

		router.GET("/authned", func(c *gin.Context) {
			// simulate authenticateRequest
			c.Set(access.RequestContextKey, access.RequestContext{
				Authenticated: access.Authenticated{
					User:         &models.Identity{Model: models.Model{ID: 12345}},
					Organization: &models.Organization{Model: models.Model{ID: 2323}},
				},
				Response: &access.Response{},
			})
		})

		return router
	}

	t.Run("identical requests are sampled", func(t *testing.T) {
		b := &bytes.Buffer{}
		router := setup(t, b)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, httptest.NewRequest("GET", "/good/1", nil))
		router.ServeHTTP(resp, httptest.NewRequest("GET", "/good/2", nil))
		router.ServeHTTP(resp, httptest.NewRequest("GET", "/good/3", nil))
		router.ServeHTTP(resp, httptest.NewRequest("GET", "/good/4", nil))
		router.ServeHTTP(resp, httptest.NewRequest("GET", "/gooder/", nil))
		router.ServeHTTP(resp, httptest.NewRequest("GET", "/good/5", nil))
		router.ServeHTTP(resp, httptest.NewRequest("POST", "/good/1", nil))
		router.ServeHTTP(resp, httptest.NewRequest("POST", "/good/2", nil))

		actual := decodeLogs(t, b)
		expected := []logEntry{
			{Method: "GET", Path: "/good/1", StatusCode: 200, Level: "info"},
			{Method: "GET", Path: "/gooder/", StatusCode: 200, Level: "info"},
			{Method: "POST", Path: "/good/1", StatusCode: 200, Level: "info"},
			{Method: "POST", Path: "/good/2", StatusCode: 200, Level: "info"},
		}
		assert.DeepEqual(t, actual, expected)
	})

	t.Run("non-200 status responses are never sampled", func(t *testing.T) {
		b := &bytes.Buffer{}
		router := setup(t, b)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, httptest.NewRequest("GET", "/bad/1", nil))
		router.ServeHTTP(resp, httptest.NewRequest("GET", "/bad/1", nil))
		router.ServeHTTP(resp, httptest.NewRequest("GET", "/bad/2", nil))
		router.ServeHTTP(resp, httptest.NewRequest("GET", "/broken", nil))
		router.ServeHTTP(resp, httptest.NewRequest("GET", "/broken", nil))
		router.ServeHTTP(resp, httptest.NewRequest("GET", "/broken", nil))

		actual := decodeLogs(t, b)
		expected := []logEntry{
			{Method: "GET", Path: "/bad/1", StatusCode: 400, Level: "info"},
			{Method: "GET", Path: "/bad/1", StatusCode: 400, Level: "info"},
			{Method: "GET", Path: "/bad/2", StatusCode: 400, Level: "info"},
			{Method: "GET", Path: "/broken", StatusCode: 500, Level: "info"},
			{Method: "GET", Path: "/broken", StatusCode: 500, Level: "info"},
			{Method: "GET", Path: "/broken", StatusCode: 500, Level: "info"},
		}
		assert.DeepEqual(t, actual, expected)
	})

	t.Run("with userID and orgID", func(t *testing.T) {
		b := &bytes.Buffer{}
		router := setup(t, b)
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, httptest.NewRequest("GET", "/authned", nil))

		actual := decodeLogs(t, b)
		expected := []logEntry{
			{
				Method:     "GET",
				Path:       "/authned",
				StatusCode: 200,
				Level:      "info",
				UserID:     uid.ID(12345),
				OrgID:      uid.ID(2323),
			},
		}
		assert.DeepEqual(t, actual, expected)
	})
}

func decodeLogs(t *testing.T, input io.Reader) []logEntry {
	const maxLogs = 15
	logs := make([]logEntry, maxLogs)
	dec := json.NewDecoder(input)
	for i := 0; i < cap(logs); i++ {
		err := dec.Decode(&logs[i])
		if errors.Is(err, io.EOF) {
			return logs[:i]
		}
		assert.NilError(t, err)
	}
	t.Errorf("more than %d logs, some were not decoded", maxLogs)
	return logs
}

type logEntry struct {
	Method     string `json:"method"`
	Path       string `json:"path"`
	StatusCode int    `json:"statusCode"`
	Level      string `json:"level"`
	UserID     uid.ID `json:"userID"`
	OrgID      uid.ID `json:"orgID"`
}
