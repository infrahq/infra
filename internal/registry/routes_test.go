package registry

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/infrahq/infra/internal/api"
	"github.com/infrahq/infra/uid"
	"github.com/stretchr/testify/require"
)

func TestBindsQuery(t *testing.T) {
	c, _ := gin.CreateTestContext(nil)

	uri, err := url.Parse("/foo?alpha=beta")
	require.NoError(t, err)
	c.Request = &http.Request{URL: uri, Method: "GET"}
	r := &struct {
		Alpha string `form:"alpha"`
	}{}
	err = bind(c, r)
	require.NoError(t, err)

	require.EqualValues(t, "beta", r.Alpha)
}

func TestBindsJSON(t *testing.T) {
	c, _ := gin.CreateTestContext(nil)

	uri, err := url.Parse("/foo")
	require.NoError(t, err)
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
	require.NoError(t, err)

	require.EqualValues(t, "zeta", r.Alpha)

}

func TestBindsUUIDs(t *testing.T) {
	c, _ := gin.CreateTestContext(nil)

	uri, err := url.Parse("/foo/e4d97df2")
	require.NoError(t, err)
	c.Request = &http.Request{URL: uri, Method: "GET"}
	c.Params = append(c.Params, gin.Param{Key: "id", Value: "e4d97df2"})
	r := &api.Resource{}
	err = bind(c, r)
	require.NoError(t, err)

	require.EqualValues(t, "e4d97df2", r.ID.String())
}

func TestBindsSnowflake(t *testing.T) {
	c, _ := gin.CreateTestContext(nil)

	id := uid.New()
	id2 := uid.New()

	uri, err := url.Parse(fmt.Sprintf("/foo/%s?form_id=%s", id.String(), id2.String()))
	require.NoError(t, err)
	c.Request = &http.Request{URL: uri, Method: "GET"}
	c.Params = append(c.Params, gin.Param{Key: "id", Value: id.String()})
	r := &struct {
		ID     uid.ID `uri:"id"`
		FormID uid.ID `form:"form_id"`
	}{}
	err = bind(c, r)
	require.NoError(t, err)

	require.Equal(t, id, r.ID)
	require.Equal(t, id2, r.FormID)
}

func TestBindsEmptyRequest(t *testing.T) {
	c, _ := gin.CreateTestContext(nil)

	uri, err := url.Parse("/foo")
	require.NoError(t, err)
	c.Request = &http.Request{URL: uri, Method: "GET"}
	r := &api.EmptyRequest{}
	err = bind(c, r)
	require.NoError(t, err)
}

func TestGetRoute(t *testing.T) {
	w := httptest.NewRecorder()
	c, e := gin.CreateTestContext(w)
	uri, _ := url.Parse("/")
	c.Request = &http.Request{
		URL: uri,
	}
	r := e.Group("/")

	get(r, "/", func(c *gin.Context, req *api.EmptyRequest) (*api.EmptyResponse, error) {
		return &api.EmptyResponse{}, nil
	})
	routes := e.Routes()

	for _, route := range routes {
		route.HandlerFunc(c)
	}

	require.EqualValues(t, http.StatusOK, w.Code)
}
