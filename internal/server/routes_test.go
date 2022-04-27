package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"gotest.tools/v3/assert"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/uid"
)

func TestBindsQuery(t *testing.T) {
	c, _ := gin.CreateTestContext(nil)

	uri, err := url.Parse("/foo?alpha=beta")
	assert.NilError(t, err)

	c.Request = &http.Request{URL: uri, Method: "GET"}
	r := &struct {
		Alpha string `form:"alpha"`
	}{}
	err = bind(c, r)
	assert.NilError(t, err)

	assert.Equal(t, "beta", r.Alpha)
}

func TestBindsJSON(t *testing.T) {
	c, _ := gin.CreateTestContext(nil)

	uri, err := url.Parse("/foo")
	assert.NilError(t, err)

	body := bytes.NewBufferString(`{"alpha": "zeta"}`)
	c.Request = &http.Request{
		URL:           uri,
		Method:        "GET",
		Body:          io.NopCloser(body),
		ContentLength: int64(body.Len()),
		Header:        http.Header{"Content-Type": []string{"application/json"}},
	}
	r := &struct {
		Alpha string `json:"alpha"`
	}{}
	err = bind(c, r)
	assert.NilError(t, err)

	assert.Equal(t, "zeta", r.Alpha)
}

func TestBindsUUIDs(t *testing.T) {
	c, _ := gin.CreateTestContext(nil)

	uri, err := url.Parse("/foo/e4d97df2")
	assert.NilError(t, err)

	c.Request = &http.Request{URL: uri, Method: "GET"}
	c.Params = append(c.Params, gin.Param{Key: "id", Value: "e4d97df2"})
	r := &api.Resource{}
	err = bind(c, r)
	assert.NilError(t, err)

	assert.Equal(t, "e4d97df2", r.ID.String())
}

func TestBindsSnowflake(t *testing.T) {
	c, _ := gin.CreateTestContext(nil)

	id := uid.New()
	id2 := uid.New()

	uri, err := url.Parse(fmt.Sprintf("/foo/%s?form_id=%s", id.String(), id2.String()))
	assert.NilError(t, err)

	c.Request = &http.Request{URL: uri, Method: "GET"}
	c.Params = append(c.Params, gin.Param{Key: "id", Value: id.String()})
	r := &struct {
		ID     uid.ID `uri:"id"`
		FormID uid.ID `form:"form_id"`
	}{}
	err = bind(c, r)
	assert.NilError(t, err)

	assert.Equal(t, id, r.ID)
	assert.Equal(t, id2, r.FormID)
}

func TestBindsEmptyRequest(t *testing.T) {
	c, _ := gin.CreateTestContext(nil)

	uri, err := url.Parse("/foo")
	assert.NilError(t, err)

	c.Request = &http.Request{URL: uri, Method: "GET"}
	r := &api.EmptyRequest{}
	err = bind(c, r)
	assert.NilError(t, err)
}

func TestGetRoute(t *testing.T) {
	w := httptest.NewRecorder()
	c, e := gin.CreateTestContext(w)
	uri, _ := url.Parse("/")
	c.Request = &http.Request{
		URL: uri,
	}
	r := e.Group("/")

	get(&API{}, r, "/", func(c *gin.Context, req *api.EmptyRequest) (*api.EmptyResponse, error) {
		return &api.EmptyResponse{}, nil
	})

	routes := e.Routes()

	for _, route := range routes {
		route.HandlerFunc(c)
	}

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestTimestampAndDurationSerialization(t *testing.T) {
	c, _ := gin.CreateTestContext(nil)

	uri, err := url.Parse("/foo")
	assert.NilError(t, err)

	orig := `{"deadline":"2022-03-23T17:50:59Z","extension":"1h35m0s"}`
	body := bytes.NewBufferString(orig)
	c.Request = &http.Request{
		URL:           uri,
		Method:        "GET",
		Body:          io.NopCloser(body),
		ContentLength: int64(body.Len()),
		Header:        http.Header{"Content-Type": []string{"application/json"}},
	}
	r := &struct {
		Deadline  api.Time     `json:"deadline"`
		Extension api.Duration `json:"extension"`
	}{}
	err = bind(c, r)
	assert.NilError(t, err)

	expected := time.Date(2022, 3, 23, 17, 50, 59, 0, time.UTC)
	assert.Equal(t, api.Time(expected), r.Deadline)
	assert.Equal(t, api.Duration(1*time.Hour+35*time.Minute), r.Extension)

	result, err := json.Marshal(r)
	assert.NilError(t, err)

	assert.Equal(t, orig, string(result))
}

func TestFullPath(t *testing.T) {
	type testCase struct {
		base     string
		path     string
		expected string
	}

	run := func(t *testing.T, tc testCase) {
		actual := fullPath(tc.base, tc.path)
		assert.Equal(t, actual, tc.expected)
	}

	testCases := []testCase{
		{base: "/", path: "/v1/something", expected: "/v1/something"},
		{base: "/", path: "v1/something", expected: "/v1/something"},
		{base: "/", path: "v1/something/:id", expected: "/v1/something/:id"},
		{base: "/", path: "/v1/something/:id", expected: "/v1/something/:id"},
		{base: "/v1", path: "something", expected: "/v1/something"},
		{base: "/v1", path: "/something", expected: "/v1/something"},
		{base: "/v1", path: "///something", expected: "/v1/something"},
		{base: "/v1", path: "/something/:id", expected: "/v1/something/:id"},
		{base: "/v1/", path: "something", expected: "/v1/something"},
		{base: "/v1/", path: "/something", expected: "/v1/something"},
		{base: "/v1/", path: "///something", expected: "/v1/something"},
		{base: "/v1/", path: "/something/:id", expected: "/v1/something/:id"},
		{base: "/v1///", path: "/something/:id", expected: "/v1/something/:id"},
	}

	for _, tc := range testCases {
		name := fmt.Sprintf("base=%v path=%v", tc.base, tc.path)
		t.Run(name, func(t *testing.T) {
			run(t, tc)
		})
	}

}
