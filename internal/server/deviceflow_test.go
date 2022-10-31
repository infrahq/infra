package server

import (
	"encoding/json"
	"fmt"
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
	srv := setupServer(t, withAdminUser)
	routes := srv.GenerateRoutes()

	org := &models.Organization{
		Name:   "foo",
		Domain: "foo.example.com",
	}
	err := data.CreateOrganization(srv.db, org)
	assert.NilError(t, err)
	user := &models.Identity{
		OrganizationMember: models.OrganizationMember{OrganizationID: org.ID},
		Name:               "joe@example.com",
	}
	err = data.CreateIdentity(srv.db, user)
	assert.NilError(t, err)

	accessKey := &models.AccessKey{
		OrganizationMember: models.OrganizationMember{OrganizationID: org.ID},
		Name:               "Foo key",
		IssuedFor:          user.ID,
		IssuedForName:      user.Name,
		ProviderID:         data.InfraProvider(srv.db).ID,
		ExpiresAt:          time.Now().Add(10 * time.Minute),
		Scopes:             models.CommaSeparatedStrings{models.ScopeAllowCreateAccessKey},
	}
	_, err = data.CreateAccessKey(srv.db, accessKey)
	assert.NilError(t, err)
	key := accessKey.Token()

	doPost := func(t *testing.T, accessKey, path string, reqObj any, respObj any) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, path, jsonBody(t, reqObj))
		req.Header.Set("Infra-Version", apiVersionLatest)
		if len(accessKey) > 0 {
			req.Header.Set("Authorization", "Bearer "+accessKey)
		}
		resp := httptest.NewRecorder()
		routes.ServeHTTP(resp, req)

		assert.Assert(t, resp.Result().StatusCode < 300, fmt.Sprintf("http status code %d: %s", resp.Result().StatusCode, resp.Body))

		if respObj != nil {
			err := json.Unmarshal(resp.Body.Bytes(), respObj)
			assert.NilError(t, err)
		}
		return resp
	}

	// start flow
	dfResp := &api.DeviceFlowResponse{}
	doPost(t, "", "http://"+org.Domain+"/api/device", api.EmptyRequest{}, dfResp)

	// get flow status pending
	pollResp := &api.DevicePollResponse{}
	doPost(t, "", "http://"+org.Domain+"/api/device/status", api.PollDeviceFlowRequest{
		DeviceCode: dfResp.DeviceCode,
	}, pollResp)

	assert.Equal(t, pollResp.Status, "pending")

	// approve
	doPost(t, key, "http://"+org.Domain+"/api/device/approve", api.ApproveDeviceFlowRequest{
		UserCode: dfResp.UserCode,
	}, nil)

	// get flow status with key
	doPost(t, "", "http://"+org.Domain+"/api/device/status", api.PollDeviceFlowRequest{
		DeviceCode: dfResp.DeviceCode,
	}, pollResp)

	assert.Equal(t, pollResp.DeviceCode, dfResp.DeviceCode)
	assert.Equal(t, pollResp.Status, "confirmed")
	newKey := pollResp.LoginResponse.AccessKey
	assert.Assert(t, len(newKey) > 0)
	assert.Assert(t, strings.Contains(newKey, "."))

	assert.Equal(t, pollResp.LoginResponse.UserID, user.ID)
}

func TestAPI_StartDeviceFlow(t *testing.T) {
	t.Run("single-tenant mode", func(t *testing.T) {
		srv := setupServer(t, withAdminUser)
		routes := srv.GenerateRoutes()

		u := "https://api.example.com:2020/api/device"
		req, err := http.NewRequest(http.MethodPost, u, nil)
		assert.NilError(t, err)
		req.Header.Set("Infra-Version", apiVersionLatest)
		resp := httptest.NewRecorder()
		routes.ServeHTTP(resp, req)

		flowResp := &api.DeviceFlowResponse{}
		err = json.NewDecoder(resp.Body).Decode(flowResp)
		assert.NilError(t, err)

		assert.Equal(t, resp.Code, http.StatusCreated, (*responseDebug)(resp))
		expected := &api.DeviceFlowResponse{
			DeviceCode:          "<any-string>",
			UserCode:            "<any-string>",
			VerificationURI:     "https://api.example.com:2020/device",
			ExpiresInSeconds:    600,
			PollIntervalSeconds: 5,
		}
		var cmpDeviceFlowResponse = gocmp.Options{
			gocmp.FilterPath(
				opt.PathField(api.DeviceFlowResponse{}, "DeviceCode"), cmpAnyString),
			gocmp.FilterPath(
				opt.PathField(api.DeviceFlowResponse{}, "UserCode"), cmpAnyString),
		}
		assert.DeepEqual(t, flowResp, expected, cmpDeviceFlowResponse)
	})

}
