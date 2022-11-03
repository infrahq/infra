package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/sync/errgroup"
	"gotest.tools/v3/assert"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

func TestReadRequest_FromQuery(t *testing.T) {
	c, _ := gin.CreateTestContext(nil)

	uri, err := url.Parse("/foo?alpha=beta")
	assert.NilError(t, err)

	c.Request = &http.Request{URL: uri, Method: "GET"}
	r := &struct {
		Alpha string `form:"alpha"`
	}{}
	err = readRequest(c, r)
	assert.NilError(t, err)

	assert.Equal(t, "beta", r.Alpha)
}

func TestReadRequest_JSON(t *testing.T) {
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
	err = readRequest(c, r)
	assert.NilError(t, err)

	assert.Equal(t, "zeta", r.Alpha)
}

func TestReadRequest_UUIDs(t *testing.T) {
	c, _ := gin.CreateTestContext(nil)

	uri, err := url.Parse("/foo/e4d97df2")
	assert.NilError(t, err)

	c.Request = &http.Request{URL: uri, Method: "GET"}
	c.Params = append(c.Params, gin.Param{Key: "id", Value: "e4d97df2"})
	r := &api.Resource{}
	err = readRequest(c, r)
	assert.NilError(t, err)

	assert.Equal(t, "e4d97df2", r.ID.String())
}

func TestReadRequest_Snowflake(t *testing.T) {
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
	err = readRequest(c, r)
	assert.NilError(t, err)

	assert.Equal(t, id, r.ID)
	assert.Equal(t, id2, r.FormID)
}

func TestReadRequest_EmptyRequest(t *testing.T) {
	c, _ := gin.CreateTestContext(nil)

	uri, err := url.Parse("/foo")
	assert.NilError(t, err)

	c.Request = &http.Request{URL: uri, Method: "GET"}
	r := &api.EmptyRequest{}
	err = readRequest(c, r)
	assert.NilError(t, err)
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
	err = readRequest(c, r)
	assert.NilError(t, err)

	expected := time.Date(2022, 3, 23, 17, 50, 59, 0, time.UTC)
	assert.Equal(t, api.Time(expected), r.Deadline)
	assert.Equal(t, api.Duration(1*time.Hour+35*time.Minute), r.Extension)

	result, err := json.Marshal(r)
	assert.NilError(t, err)

	assert.Equal(t, orig, string(result))
}

func TestTrimWhitespace(t *testing.T) {
	srv := setupServer(t, withAdminUser)
	routes := srv.GenerateRoutes()

	userID := uid.New()
	// nolint:noctx
	req := httptest.NewRequest(http.MethodPost, "/api/grants", jsonBody(t, api.GrantRequest{
		User:      userID,
		Privilege: "admin   ",
		Resource:  " kubernetes.production.*",
	}))
	req.Header.Add("Authorization", "Bearer "+adminAccessKey(srv))
	req.Header.Add("Infra-Version", "0.13.1")

	resp := httptest.NewRecorder()
	routes.ServeHTTP(resp, req)
	assert.Equal(t, resp.Code, http.StatusCreated, resp.Body.String())

	// nolint:noctx
	req = httptest.NewRequest(http.MethodGet, "/api/grants?privilege=%20admin%20&user_id="+userID.String(), nil)
	req.Header.Add("Authorization", "Bearer "+adminAccessKey(srv))
	req.Header.Add("Infra-Version", "0.13.1")

	resp = httptest.NewRecorder()
	routes.ServeHTTP(resp, req)
	assert.Equal(t, resp.Code, http.StatusOK)

	rb := &api.ListResponse[api.Grant]{}
	err := json.Unmarshal(resp.Body.Bytes(), rb)
	assert.NilError(t, err)

	assert.Equal(t, len(rb.Items), 2, rb.Items)
	expected := api.Grant{
		User:      userID,
		Privilege: "admin",
		Resource:  "kubernetes.production.*",
	}
	assert.DeepEqual(t, rb.Items[1], expected, cmpAPIGrantShallow)
}

func TestWrapRoute_TxnRollbackOnError(t *testing.T) {
	srv := setupServer(t)
	router := gin.New()

	r := route[api.EmptyRequest, *api.EmptyResponse]{
		handler: func(c *gin.Context, request *api.EmptyRequest) (*api.EmptyResponse, error) {
			rCtx := getRequestContext(c)

			user := &models.Identity{
				Model:              models.Model{ID: 1555},
				Name:               "user@example.com",
				OrganizationMember: models.OrganizationMember{OrganizationID: srv.db.DefaultOrg.ID},
			}
			if err := data.CreateIdentity(rCtx.DBTxn, user); err != nil {
				return nil, err
			}

			return nil, fmt.Errorf("this failed")
		},
		routeSettings: routeSettings{
			infraVersionHeaderOptional: true,
			authenticationOptional:     true,
			organizationOptional:       true,
		},
	}

	api := &API{server: srv}
	add(api, rg(router.Group("/")), "POST", "/do", r)

	resp := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/do", nil)
	router.ServeHTTP(resp, req)

	assert.Equal(t, resp.Code, http.StatusInternalServerError)

	// The user should not exist, because the txn was rollbed back
	_, err := data.GetIdentity(srv.db, data.GetIdentityOptions{ByID: uid.ID(1555)})
	assert.ErrorIs(t, err, internal.ErrNotFound)
}

func TestWrapRoute_HandleErrorOnCommit(t *testing.T) {
	srv := setupServer(t)
	router := gin.New()

	r := route[api.EmptyRequest, *api.EmptyResponse]{
		handler: func(c *gin.Context, request *api.EmptyRequest) (*api.EmptyResponse, error) {
			rCtx := getRequestContext(c)

			// Commit the transaction so that the call in wrapRoute returns an error
			err := rCtx.DBTxn.Commit()
			return nil, err
		},
		routeSettings: routeSettings{
			infraVersionHeaderOptional: true,
			authenticationOptional:     true,
			organizationOptional:       true,
		},
	}

	api := &API{server: srv}
	add(api, rg(router.Group("/")), "POST", "/do", r)

	resp := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/do", nil)
	router.ServeHTTP(resp, req)

	assert.Equal(t, resp.Code, http.StatusInternalServerError)
}

func TestInfraVersionHeader(t *testing.T) {
	srv := setupServer(t, withAdminUser)
	routes := srv.GenerateRoutes()

	body := jsonBody(t, api.CreateUserRequest{Name: "usera@example.com"})
	// nolint:noctx
	req := httptest.NewRequest(http.MethodPost, "/api/users", body)
	req.Header.Set("Authorization", "Bearer "+adminAccessKey(srv))

	resp := httptest.NewRecorder()
	routes.ServeHTTP(resp, req)

	assert.Equal(t, resp.Code, http.StatusBadRequest, resp.Body.String())

	respBody := &api.Error{}
	err := json.Unmarshal(resp.Body.Bytes(), respBody)
	assert.NilError(t, err)

	assert.Assert(t, strings.Contains(respBody.Message, "Infra-Version header is required"), respBody.Message)
}

var apiVersionLatest = internal.FullVersion()

func TestRequestTimeout(t *testing.T) {
	if testing.Short() {
		t.Skip("too slow for short run")
	}
	srv := setupServer(t)
	srv.options.API.RequestTimeout = time.Second
	routes := srv.GenerateRoutes()
	router, ok := routes.Handler.(*gin.Engine)
	assert.Assert(t, ok)
	a := &API{server: srv}

	group := &routeGroup{RouterGroup: router.Group("/"), noAuthentication: true, noOrgRequired: true}
	add(a, group, http.MethodGet, "/sleep", route[api.EmptyRequest, *api.EmptyResponse]{
		handler: func(c *gin.Context, req *api.EmptyRequest) (*api.EmptyResponse, error) {
			ctx := getRequestContext(c)

			_, exist := ctx.Request.Context().Deadline()
			assert.Assert(t, exist)

			_, err := ctx.DBTxn.Exec("select pg_sleep(2)")
			assert.Error(t, err, "timeout: context deadline exceeded", "expected this query to time out and get cancelled")

			return nil, err
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/sleep", nil)
	req.Header.Add("Infra-Version", "0.13.1")

	resp := httptest.NewRecorder()
	started := time.Now()
	routes.ServeHTTP(resp, req)
	elapsed := time.Since(started)

	assert.Equal(t, resp.Code, http.StatusGatewayTimeout, resp.Body.String())

	assert.Assert(t, elapsed < 1500*time.Millisecond, "expected request to time out due to the timeout context, but it did not")
}

func TestGenerateRoutes_OneRequestDoesNotBlockOthers(t *testing.T) {
	withShortRequestTimeout := func(t *testing.T, options *Options) {
		options.API.RequestTimeout = 250 * time.Millisecond
		options.API.BlockingRequestTimeout = 1500 * time.Millisecond
	}
	srv := setupServer(t, withAdminUser, withShortRequestTimeout)
	routes := srv.GenerateRoutes()

	g := errgroup.Group{}
	// start a blocking request in the background
	g.Go(func() error {
		urlPath := "/api/grants?destination=infra&lastUpdateIndex=10001"
		req := httptest.NewRequest(http.MethodGet, urlPath, nil)
		req.Header.Set("Authorization", "Bearer "+adminAccessKey(srv))
		req.Header.Add("Infra-Version", apiVersionLatest)

		resp := httptest.NewRecorder()
		routes.ServeHTTP(resp, req)
		return nil
	})

	// perform many short-lived requests with the same user
	start := time.Now()
	var count int
	for time.Since(start) < srv.options.API.BlockingRequestTimeout && count < 3 {
		urlPath := "/api/grants?destination=infra"
		req := httptest.NewRequest(http.MethodGet, urlPath, nil)
		req.Header.Set("Authorization", "Bearer "+adminAccessKey(srv))
		req.Header.Add("Infra-Version", apiVersionLatest)

		resp := httptest.NewRecorder()
		routes.ServeHTTP(resp, req)
		assert.Equal(t, resp.Code, http.StatusOK, resp.Body.String())
		count++
	}

	assert.NilError(t, g.Wait())
	// The count is likely close to 40, but use a low threshold to prevent flakes.
	// Anything more than 2 should indicate the requests did not block each other.
	assert.Assert(t, count >= 3, "count=%d", count)
}
