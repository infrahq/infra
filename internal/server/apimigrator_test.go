package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"gotest.tools/v3/assert"

	"github.com/infrahq/infra/api"
)

type legacyTestRequest struct {
	CucumberCount int `form:"cucumberCount"`
	CarrotCount   int `form:"carrotCount"`
}

type upgradedTestRequest struct {
	VegetableCount int `form:"vegetableCount"`
}

func TestAddRequestRewrite(t *testing.T) {
	srv := setupServer(t, withAdminIdentity)

	a := &API{server: srv}
	router := gin.New()

	addRequestRewrite(a, "get", "/test", "0.1.0", func(old legacyTestRequest) upgradedTestRequest {
		return upgradedTestRequest{
			VegetableCount: old.CarrotCount + old.CucumberCount,
		}
	})

	get(a, router.Group("/"), "/test", func(c *gin.Context, req *upgradedTestRequest) (*api.EmptyResponse, error) {
		assert.Assert(t, req.VegetableCount == 12)
		return nil, nil
	})

	resp := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test?cucumberCount=5&carrotCount=7", nil)
	router.ServeHTTP(resp, req)

	assert.Assert(t, resp.Result().StatusCode == 200)
}

func TestRedirect(t *testing.T) {
	srv := setupServer(t, withAdminIdentity)

	a := &API{server: srv}
	router := gin.New()

	addRedirect(a, http.MethodGet, "/test", "/supertest", "0.1.0")

	get(a, router.Group("/"), "/supertest", func(c *gin.Context, req *upgradedTestRequest) (*api.EmptyResponse, error) {
		assert.Assert(t, req.VegetableCount == 17)
		return nil, nil
	})

	resp := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test?vegetableCount=17", nil)
	router.ServeHTTP(resp, req)

	assert.Assert(t, resp.Result().StatusCode == 200)
}
