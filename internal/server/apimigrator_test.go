package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Masterminds/semver/v3"
	"github.com/gin-gonic/gin"
	"gotest.tools/v3/assert"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/uid"
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

	a := &API{server: srv}
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
	srv := setupServer(t, withAdminUser)

	a := &API{server: srv}
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

func TestRedirectOfRequestAndResponseRewrite(t *testing.T) {
	srv := setupServer(t, withAdminUser)

	a := &API{server: srv}
	router := gin.New()

	addRedirect(a, "get", "/oldtest", "/test", "0.1.0")

	addRequestRewrite(a, "get", "/test", "0.1.1", func(old legacyTestRequest) upgradedTestRequest {
		return upgradedTestRequest{
			VegetableCount: old.CarrotCount + old.CucumberCount,
		}
	})

	addResponseRewrite(a, "get", "/test", "0.1.1", func(ur upgradedResponse) legacyResponse {
		return legacyResponse{
			Shoes: ur.Loafers + ur.Sneakers,
		}
	})

	get(a, router.Group("/"), "/test", func(c *gin.Context, req *upgradedTestRequest) (*upgradedResponse, error) {
		assert.Equal(t, req.VegetableCount, 12)

		return &upgradedResponse{
			Loafers:  5,
			Sneakers: 3,
		}, nil
	})

	resp := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/oldtest?cucumberCount=5&carrotCount=7", nil)
	router.ServeHTTP(resp, req)

	assert.Equal(t, resp.Result().StatusCode, 200)

	lr := &legacyResponse{}
	err := json.Unmarshal(resp.Body.Bytes(), lr)
	assert.NilError(t, err)
	assert.Equal(t, lr.Shoes, 8)
}

func TestRedirectOfRequestAndResponseRewriteWithStackedRedirects(t *testing.T) {
	srv := setupServer(t, withAdminUser)

	a := &API{server: srv}
	router := gin.New()

	addRequestRewrite(a, "get", "/test", "0.1.1", func(old legacyTestRequest) upgradedTestRequest {
		return upgradedTestRequest{
			VegetableCount: old.CarrotCount + old.CucumberCount,
		}
	})

	addResponseRewrite(a, "get", "/test", "0.1.1", func(ur upgradedResponse) legacyResponse {
		return legacyResponse{
			Shoes: ur.Loafers + ur.Sneakers,
		}
	})

	addRedirect(a, "get", "/test", "/superbettertest", "0.1.2")
	addRedirect(a, "get", "/superbettertest", "/awesometest", "0.1.3")

	get(a, router.Group("/"), "/awesometest", func(c *gin.Context, req *upgradedTestRequest) (*upgradedResponse, error) {
		assert.Equal(t, req.VegetableCount, 12)

		return &upgradedResponse{
			Loafers:  5,
			Sneakers: 3,
		}, nil
	})

	t.Run("v0.1.1", func(t *testing.T) {
		resp := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/test?cucumberCount=5&carrotCount=7", nil)
		req.Header.Add("Infra-Version", "0.1.1")
		router.ServeHTTP(resp, req)

		assert.Equal(t, resp.Result().StatusCode, 200)

		lr := &legacyResponse{}
		err := json.Unmarshal(resp.Body.Bytes(), lr)
		assert.NilError(t, err)
		assert.Equal(t, lr.Shoes, 8)
	})
	t.Run("v0.1.2", func(t *testing.T) {
		resp := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/superbettertest?vegetableCount=12", nil)
		req.Header.Add("Infra-Version", "0.1.2")
		router.ServeHTTP(resp, req)

		assert.Equal(t, resp.Result().StatusCode, 200)

		lr := &upgradedResponse{}
		err := json.Unmarshal(resp.Body.Bytes(), lr)
		assert.NilError(t, err)
		assert.Equal(t, lr.Loafers, 5)
		assert.Equal(t, lr.Sneakers, 3)
	})
	t.Run("v0.1.3", func(t *testing.T) {
		resp := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/awesometest?vegetableCount=12", nil)
		req.Header.Add("Infra-Version", "0.1.3")
		router.ServeHTTP(resp, req)

		assert.Equal(t, resp.Result().StatusCode, 200)

		lr := &upgradedResponse{}
		err := json.Unmarshal(resp.Body.Bytes(), lr)
		assert.NilError(t, err)
		assert.Equal(t, lr.Loafers, 5)
		assert.Equal(t, lr.Sneakers, 3)
	})
	t.Run("living in the past: select the 0.1.3 path for v0.1.4", func(t *testing.T) {
		resp := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/superbettertest?vegetableCount=12", nil)
		req.Header.Add("Infra-Version", "0.1.4")
		router.ServeHTTP(resp, req)

		assert.Equal(t, resp.Result().StatusCode, 404)
	})
	t.Run("living in the future: select the 0.1.3 path for v0.1.2", func(t *testing.T) {
		resp := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/awesometest?vegetableCount=12", nil)
		req.Header.Add("Infra-Version", "0.1.2")
		router.ServeHTTP(resp, req)

		// I guess this is okay.
		// As long as the client knows how to handle the request this will work.
		assert.Equal(t, resp.Result().StatusCode, 200)
	})
}

func TestRewriteOfRedirectedRoute(t *testing.T) {
	srv := setupServer(t, withAdminUser)

	a := &API{server: srv}
	router := gin.New()

	addRedirect(a, "get", "/test", "/awesometest", "0.1.3")

	addRequestRewrite(a, "get", "/awesometest", "0.1.4", func(old legacyTestRequest) upgradedTestRequest {
		return upgradedTestRequest{
			VegetableCount: old.CarrotCount + old.CucumberCount,
		}
	})

	addResponseRewrite(a, "get", "/awesometest", "0.1.4", func(ur upgradedResponse) legacyResponse {
		return legacyResponse{
			Shoes: ur.Loafers + ur.Sneakers,
		}
	})

	get(a, router.Group("/"), "/awesometest", func(c *gin.Context, req *upgradedTestRequest) (*upgradedResponse, error) {
		assert.Equal(t, req.VegetableCount, 12)

		return &upgradedResponse{
			Loafers:  5,
			Sneakers: 3,
		}, nil
	})

	t.Run("v0.1.3", func(t *testing.T) {
		resp := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/test?cucumberCount=5&carrotCount=7", nil)
		req.Header.Add("Infra-Version", "0.1.3")
		router.ServeHTTP(resp, req)

		assert.Equal(t, resp.Result().StatusCode, 200)

		lr := &legacyResponse{}
		err := json.Unmarshal(resp.Body.Bytes(), lr)
		assert.NilError(t, err)
		assert.Equal(t, lr.Shoes, 8)
	})
	t.Run("v0.1.4", func(t *testing.T) {
		resp := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/awesometest?cucumberCount=5&carrotCount=7", nil)
		req.Header.Add("Infra-Version", "0.1.4")
		router.ServeHTTP(resp, req)

		assert.Equal(t, resp.Result().StatusCode, 200)

		lr := &legacyResponse{}
		err := json.Unmarshal(resp.Body.Bytes(), lr)
		assert.NilError(t, err)
		assert.Equal(t, lr.Shoes, 8)
	})
	t.Run("v0.1.5", func(t *testing.T) {
		resp := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/awesometest?vegetableCount=12", nil)
		req.Header.Add("Infra-Version", "0.1.5")
		router.ServeHTTP(resp, req)

		assert.Equal(t, resp.Result().StatusCode, 200)

		lr := &upgradedResponse{}
		err := json.Unmarshal(resp.Body.Bytes(), lr)
		assert.NilError(t, err)
		assert.Equal(t, lr.Loafers, 5)
	})
}

func TestRedirectWithPathVariable(t *testing.T) {
	srv := setupServer(t, withAdminUser)

	a := &API{server: srv}
	router := gin.New()

	type getUserRequest struct {
		ID uid.ID `uri:"id"`
	}
	id := uid.New()
	addRedirect(a, "get", "/identity/:id", "/user/:id", "0.1.0")

	get(a, router.Group("/"), "/user/:id", func(c *gin.Context, req *getUserRequest) (*api.EmptyResponse, error) {
		assert.Equal(t, req.ID, id)

		return nil, nil
	})

	resp := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/identity/"+id.String(), nil)
	router.ServeHTTP(resp, req)

	assert.Equal(t, resp.Result().StatusCode, 200)
}

type legacyResponse struct {
	Shoes int
}

type upgradedResponse struct {
	Loafers  int `json:"loafers"`
	Sneakers int `json:"sneakers,omitempty"`
}

func TestAddResponseRewrite(t *testing.T) {
	srv := setupServer(t, withAdminUser)

	a := &API{server: srv}
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
	srv := setupServer(t, withAdminUser)

	a := &API{server: srv}
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

func TestEmptyVersionHeader(t *testing.T) {
	srv := setupServer(t, withAdminUser)

	a := &API{server: srv}
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

	resp := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Add("Infra-Version", "")
	router.ServeHTTP(resp, req)

	if afterVersion("0.14.0") {
		assert.Equal(t, resp.Result().StatusCode, http.StatusBadRequest, "Request should fail: Client must provide Infra-Version")

		apiErr := &api.Error{}
		err := json.Unmarshal(resp.Body.Bytes(), apiErr)
		assert.NilError(t, err)
		assert.Assert(t, strings.Contains(apiErr.Message, "Infra-Version header required"))

	} else {
		assert.Equal(t, resp.Result().StatusCode, 200)

		r := &legacyResponse{}
		err := json.Unmarshal(resp.Body.Bytes(), r)
		assert.NilError(t, err)
		assert.Equal(t, r.Shoes, 8)
	}
}

func TestRedirectWithARequestRewriteToQueryParameter(t *testing.T) {
	srv := setupServer(t, withAdminUser)

	a := &API{server: srv}
	router := gin.New()

	addRedirect(a, "get", "/oldtest/:id", "/awesometest", "0.1.3", func(c *gin.Context) {
		id := c.Param("id")
		q := c.Request.URL.Query()
		q.Add("vegetableCount", id)
		c.Request.URL.RawQuery = q.Encode()
		c.Next()
	})

	get(a, router.Group("/"), "/awesometest", func(c *gin.Context, req *upgradedTestRequest) (*upgradedResponse, error) {
		assert.Equal(t, req.VegetableCount, 152)

		return &upgradedResponse{
			Loafers:  5,
			Sneakers: 3,
		}, nil
	})

	t.Run("v0.1.2", func(t *testing.T) {
		resp := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/oldtest/152", nil)
		req.Header.Add("Infra-Version", "0.1.2")
		router.ServeHTTP(resp, req)

		assert.Equal(t, resp.Result().StatusCode, 200)

		lr := &upgradedResponse{}
		err := json.Unmarshal(resp.Body.Bytes(), lr)
		assert.NilError(t, err)
		assert.Equal(t, lr.Loafers, 5)
	})

}

func afterVersion(ver string) bool {
	serverVer, _ := semver.NewVersion(internal.FullVersion())
	checkVer, _ := semver.NewVersion(ver)
	return !serverVer.LessThan(checkVer)
}
