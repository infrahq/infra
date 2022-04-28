package server

import (
	"encoding/json"
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
	srv := setupServer(t, withAdminUser)

	a := &API{server: srv, disableOpenAPIGeneration: true}
	router := gin.New()

	addRequestRewrite(a, "get", "/test", "0.1.0", func(old legacyTestRequest) upgradedTestRequest {
		return upgradedTestRequest{
			VegetableCount: old.CarrotCount + old.CucumberCount,
		}
	})

	get(a, router.Group("/"), "/test", func(c *gin.Context, req *upgradedTestRequest) (*api.EmptyResponse, error) {
		assert.Equal(t, req.VegetableCount, 12)
		return nil, nil
	})

	resp := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test?cucumberCount=5&carrotCount=7", nil)
	router.ServeHTTP(resp, req)

	assert.Equal(t, resp.Result().StatusCode, 200)
}

func TestStackedAddRequestRewrite(t *testing.T) {
	srv := setupServer(t, withAdminIdentity)

	a := &API{server: srv, disableOpenAPIGeneration: true}
	router := gin.New()

	addRequestRewrite(a, "get", "/test", "0.1.0", func(old legacyTestRequest) upgradedTestRequest {
		return upgradedTestRequest{
			VegetableCount: old.CarrotCount + old.CucumberCount,
		}
	})

	addRequestRewrite(a, "get", "/test", "0.1.1", func(old upgradedTestRequest) upgradedTestRequest {
		return upgradedTestRequest{
			VegetableCount: old.VegetableCount * 2,
		}
	})

	get(a, router.Group("/"), "/test", func(c *gin.Context, req *upgradedTestRequest) (*api.EmptyResponse, error) {
		assert.Equal(t, req.VegetableCount, 24)
		return nil, nil
	})

	resp := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test?cucumberCount=5&carrotCount=7", nil)
	router.ServeHTTP(resp, req)

	assert.Equal(t, resp.Result().StatusCode, 200)
}

func TestRedirect(t *testing.T) {
	srv := setupServer(t, withAdminUser)

	a := &API{server: srv, disableOpenAPIGeneration: true}
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

type legacyResponse struct {
	Shoes int
}

type upgradedResponse struct {
	Loafers  int
	Sneakers int
}

func TestAddResponseRewrite(t *testing.T) {
	srv := setupServer(t, withAdminIdentity)

	a := &API{server: srv, disableOpenAPIGeneration: true}
	router := gin.New()

	addResponseRewrite(a, "get", "/test", "0.1.0", func(n upgradedResponse) legacyResponse {
		return legacyResponse{
			Shoes: n.Loafers + n.Sneakers,
		}
	})

	get(a, router.Group("/"), "/test", func(c *gin.Context, _ *api.EmptyRequest) (*upgradedResponse, error) {
		return &upgradedResponse{
			Loafers:  3,
			Sneakers: 5,
		}, nil
	})

	t.Run("old version downgrades", func(t *testing.T) {
		resp := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Add("Infra-Version", "0.1.0")
		router.ServeHTTP(resp, req)

		assert.Equal(t, resp.Result().StatusCode, 200)

		r := &legacyResponse{}
		err := json.Unmarshal(resp.Body.Bytes(), r)
		assert.NilError(t, err)
		assert.Equal(t, r.Shoes, 8)
	})

	t.Run("new version unchanged", func(t *testing.T) {
		resp := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Add("Infra-Version", "0.1.1")
		router.ServeHTTP(resp, req)

		assert.Equal(t, resp.Result().StatusCode, 200)

		r := &upgradedResponse{}
		err := json.Unmarshal(resp.Body.Bytes(), r)
		assert.NilError(t, err)
		assert.Equal(t, r.Loafers, 3)
		assert.Equal(t, r.Sneakers, 5)
	})
}

func TestStackedResponseRewrites(t *testing.T) {
	srv := setupServer(t, withAdminIdentity)

	a := &API{server: srv, disableOpenAPIGeneration: true}
	router := gin.New()

	addResponseRewrite(a, "get", "/test", "0.1.0", func(n upgradedResponse) legacyResponse {
		return legacyResponse{
			Shoes: n.Loafers + n.Sneakers,
		}
	})

	addResponseRewrite(a, "get", "/test", "0.1.1", func(n upgradedResponse) upgradedResponse {
		return upgradedResponse{
			Loafers:  n.Loafers * 2,
			Sneakers: n.Sneakers * 2,
		}
	})

	get(a, router.Group("/"), "/test", func(c *gin.Context, _ *api.EmptyRequest) (*upgradedResponse, error) {
		return &upgradedResponse{
			Loafers:  3,
			Sneakers: 5,
		}, nil
	})

	resp := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Add("Infra-Version", "0.1.0")
	router.ServeHTTP(resp, req)

	assert.Equal(t, resp.Result().StatusCode, 200)

	r := &legacyResponse{}
	err := json.Unmarshal(resp.Body.Bytes(), r)
	assert.NilError(t, err)
	assert.Equal(t, r.Shoes, 16)

}
