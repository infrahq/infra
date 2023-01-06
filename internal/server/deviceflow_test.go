package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	gocmp "github.com/google/go-cmp/cmp"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/opt"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
)

func TestDeviceFlow(t *testing.T) {
	srv := setupServer(t, withAdminUser, withMultiOrgEnabled)
	routes := srv.GenerateRoutes()

	org := &models.Organization{
		Name:   "foo",
		Domain: "foo.example.com",
	}

	err := data.CreateOrganization(srv.db, org)
	assert.NilError(t, err)

	tx := txnForTestCase(t, srv.db, org.ID)

	user := &models.Identity{
		OrganizationMember: models.OrganizationMember{OrganizationID: org.ID},
		Name:               "joe@example.com",
	}

	err = data.CreateIdentity(tx, user)
	assert.NilError(t, err)

	expires := time.Now().Add(10 * time.Minute).UTC().Truncate(time.Second)

	scoped := &models.AccessKey{
		OrganizationMember: models.OrganizationMember{OrganizationID: org.ID},
		Name:               "Scoped",
		IssuedFor:          user.ID,
		IssuedForName:      user.Name,
		ProviderID:         data.InfraProvider(tx).ID,
		ExpiresAt:          expires,
		Scopes:             models.CommaSeparatedStrings{models.ScopeAllowCreateAccessKey},
	}

	notscoped := &models.AccessKey{
		OrganizationMember: models.OrganizationMember{OrganizationID: org.ID},
		Name:               "Not scoped",
		IssuedFor:          user.ID,
		IssuedForName:      user.Name,
		ProviderID:         data.InfraProvider(tx).ID,
		ExpiresAt:          expires,
	}

	_, err = data.CreateAccessKey(tx, scoped)
	assert.NilError(t, err)

	_, err = data.CreateAccessKey(tx, notscoped)
	assert.NilError(t, err)

	assert.NilError(t, tx.Commit())

	key := scoped.Token()
	keyNotscoped := notscoped.Token()

	request := func(t *testing.T, method, uri, accessKey string, reqObj any, respObj any) *httptest.ResponseRecorder {
		t.Helper()
		var body io.Reader
		if reqObj != nil {
			body = jsonBody(t, reqObj)
		}
		req := httptest.NewRequest(method, uri, body)
		req.Header.Set("Infra-Version", apiVersionLatest)
		if len(accessKey) > 0 {
			req.Header.Set("Authorization", "Bearer "+accessKey)
		}
		resp := httptest.NewRecorder()
		routes.ServeHTTP(resp, req)

		if respObj != nil {
			err := json.Unmarshal(resp.Body.Bytes(), respObj)
			assert.NilError(t, err)
		}
		return resp
	}

	// start flow
	dfResp := &api.DeviceFlowResponse{}
	resp := request(t, "POST", "http://"+org.Domain+"/api/device", "", api.EmptyRequest{}, dfResp)
	assert.Assert(t, resp.Result().StatusCode < 300, fmt.Sprintf("http status code %d: %s", resp.Result().StatusCode, resp.Body))

	// get flow status pending
	statusResp := &api.DeviceFlowStatusResponse{}
	resp = request(t, "POST", "http://"+org.Domain+"/api/device/status", "", api.DeviceFlowStatusRequest{
		DeviceCode: dfResp.DeviceCode,
	}, statusResp)
	assert.Assert(t, resp.Result().StatusCode < 300, fmt.Sprintf("http status code %d: %s", resp.Result().StatusCode, resp.Body))

	assert.Equal(t, statusResp.Status, api.DeviceFlowStatusPending)

	// approve with no scope
	resp = request(t, "POST", "http://"+org.Domain+"/api/device/approve", keyNotscoped, api.ApproveDeviceFlowRequest{
		UserCode: dfResp.UserCode,
	}, nil)
	assert.Equal(t, resp.Result().StatusCode, http.StatusForbidden, (*responseDebug)(resp))

	// approve with scopes
	resp = request(t, "POST", "http://"+org.Domain+"/api/device/approve", key, api.ApproveDeviceFlowRequest{
		UserCode: dfResp.UserCode,
	}, nil)
	assert.Assert(t, resp.Result().StatusCode < 300,
		fmt.Sprintf("http status code %d: %s", resp.Result().StatusCode, resp.Body))

	// approve again does nothing
	resp = request(t, "POST", "http://"+org.Domain+"/api/device/approve", key, api.ApproveDeviceFlowRequest{
		UserCode: dfResp.UserCode,
	}, nil)
	assert.Assert(t, resp.Result().StatusCode == 201, fmt.Sprintf("http status code %d: %s", resp.Result().StatusCode, resp.Body))

	// get flow status with key
	resp = request(t, "POST", "http://"+org.Domain+"/api/device/status", "", api.DeviceFlowStatusRequest{
		DeviceCode: dfResp.DeviceCode,
	}, statusResp)
	assert.Assert(t, resp.Result().StatusCode < 300, fmt.Sprintf("http status code %d: %s", resp.Result().StatusCode, resp.Body))
	assert.Equal(t, statusResp.Status, api.DeviceFlowStatusConfirmed)
	assert.Equal(t, statusResp.DeviceCode, dfResp.DeviceCode)
	assert.Equal(t, statusResp.LoginResponse.Name, user.Name)
	assert.Equal(t, statusResp.LoginResponse.UserID, user.ID)
	assert.Equal(t, statusResp.LoginResponse.OrganizationName, org.Name)

	// Verify access key is valid
	var loginuser api.User
	_ = request(t, "GET", "http://"+org.Domain+"/api/users/self", statusResp.LoginResponse.AccessKey, nil, &loginuser)
	assert.Equal(t, loginuser.Name, user.Name)

	// valid access key
	newKey := statusResp.LoginResponse.AccessKey
	assert.Assert(t, len(newKey) > 0)
	assert.Assert(t, strings.Contains(newKey, "."))

	// Approving again after fulfilling the request should fail
	resp = request(t, "POST", "http://"+org.Domain+"/api/device/approve", key, api.ApproveDeviceFlowRequest{
		UserCode: dfResp.UserCode,
	}, nil)
	assert.Assert(t, resp.Result().StatusCode == 404, fmt.Sprintf("http status code %d: %s", resp.Result().StatusCode, resp.Body))

	// Status check should fail now that the request is fulfilled
	resp = request(t, "POST", "http://"+org.Domain+"/api/device/status", "", api.DeviceFlowStatusRequest{
		DeviceCode: dfResp.DeviceCode,
	}, statusResp)
	assert.Assert(t, resp.Result().StatusCode == 401, fmt.Sprintf("http status code %d: %s", resp.Result().StatusCode, resp.Body))
}

func TestAPI_StartDeviceFlow(t *testing.T) {
	t.Run("single-tenant mode", func(t *testing.T) {
		srv := setupServer(t, withAdminUser)
		routes := srv.GenerateRoutes()

		req := httptest.NewRequest(http.MethodPost, "https://api.example.com:2020/api/device", nil)
		req.Header.Set("Infra-Version", apiVersionLatest)
		resp := httptest.NewRecorder()
		routes.ServeHTTP(resp, req)

		flowResp := &api.DeviceFlowResponse{}
		err := json.NewDecoder(resp.Body).Decode(flowResp)
		assert.NilError(t, err)

		assert.Equal(t, resp.Code, http.StatusCreated, (*responseDebug)(resp))
		expected := &api.DeviceFlowResponse{
			DeviceCode:          "<any-string>",
			UserCode:            "<any-string>",
			VerificationURI:     "https://api.example.com:2020/device",
			ExpiresInSeconds:    600,
			PollIntervalSeconds: 5,
		}
		cmpDeviceFlowResponse := gocmp.Options{
			gocmp.FilterPath(
				opt.PathField(api.DeviceFlowResponse{}, "DeviceCode"), cmpAnyString),
			gocmp.FilterPath(
				opt.PathField(api.DeviceFlowResponse{}, "UserCode"), cmpAnyString),
		}
		assert.DeepEqual(t, flowResp, expected, cmpDeviceFlowResponse)
	})

	t.Run("non-existent org", func(t *testing.T) {
		srv := setupServer(t, withAdminUser, func(t *testing.T, opts *Options) {
			opts.EnableSignup = true
			opts.BaseDomain = "example.com"
			opts.DefaultOrganizationDomain = "example.example.com"
		})
		routes := srv.GenerateRoutes()

		req := httptest.NewRequest(http.MethodPost, "https://nonexistent-org.example.com:2020/api/device", nil)
		req.Header.Set("Infra-Version", apiVersionLatest)
		resp := httptest.NewRecorder()
		routes.ServeHTTP(resp, req)

		flowResp := &api.DeviceFlowResponse{}
		err := json.NewDecoder(resp.Body).Decode(flowResp)
		assert.NilError(t, err)

		assert.Equal(t, resp.Code, http.StatusCreated, (*responseDebug)(resp))
		expected := &api.DeviceFlowResponse{
			DeviceCode:          "<any-string>",
			UserCode:            "<any-string>",
			VerificationURI:     "https://nonexistent-org.example.com:2020/device",
			ExpiresInSeconds:    600,
			PollIntervalSeconds: 5,
		}
		cmpDeviceFlowResponse := gocmp.Options{
			gocmp.FilterPath(
				opt.PathField(api.DeviceFlowResponse{}, "DeviceCode"), cmpAnyString),
			gocmp.FilterPath(
				opt.PathField(api.DeviceFlowResponse{}, "UserCode"), cmpAnyString),
		}
		assert.DeepEqual(t, flowResp, expected, cmpDeviceFlowResponse)
	})
}
