package registry

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestRequestTimeoutError(t *testing.T) {
	requestTimeout = 100 * time.Millisecond

	router := gin.New()
	router.Use(RequestTimeoutMiddleware)
	router.GET("/", func(c *gin.Context) {
		time.Sleep(110 * time.Millisecond)

		require.Error(t, c.Request.Context().Err())

		c.Status(200)
	})
	router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil))
}

func TestRequestTimeoutSuccess(t *testing.T) {
	requestTimeout = 60 * time.Second

	router := gin.New()
	router.Use(RequestTimeoutMiddleware)
	router.GET("/", func(c *gin.Context) {
		require.NoError(t, c.Request.Context().Err())

		c.Status(200)
	})
	router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil))
}
