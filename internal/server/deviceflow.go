package server

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/access"
	"github.com/infrahq/infra/internal/generate"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
)

const DeviceCodeExpirySeconds = 600

func (a *API) StartDeviceFlow(c *gin.Context, req *api.EmptyRequest) (*api.DeviceFlowResponse, error) {
	rctx := getRequestContext(c)
	tries := 0
retry:
	tries++
	userCode, err := generate.CryptoRandom(8, generate.CharsetDeviceFlowUserCode)
	if err != nil {
		return nil, err
	}

	deviceCode, err := generate.CryptoRandom(38, generate.CharsetAlphaNumeric)
	if err != nil {
		return nil, err
	}

	err = access.CreateDeviceFlowAuthRequest(rctx, &models.DeviceFlowAuthRequest{
		UserCode:   userCode,
		DeviceCode: deviceCode,
		ExpiresAt:  time.Now().Add(DeviceCodeExpirySeconds * time.Second),
	})
	if err != nil {
		if tries < 10 && errors.Is(err, &data.UniqueConstraintError{}) {
			goto retry
		}
		return nil, err
	}

	host := a.server.options.BaseDomain
	if rctx.Authenticated.Organization != nil {
		host = rctx.Authenticated.Organization.Domain
	}
	if host == "" {
		// Default to the request hostname when in single tenant mode
		host = rctx.Request.URL.Host
	}

	return &api.DeviceFlowResponse{
		DeviceCode:          deviceCode,
		VerificationURI:     fmt.Sprintf("https://%s/device", host),
		UserCode:            userCode[0:4] + "-" + userCode[4:],
		ExpiresInSeconds:    DeviceCodeExpirySeconds,
		PollIntervalSeconds: 5,
	}, nil
}

// GetDeviceFlowStatus is an API handler for checking the status of a device
// flow login. The response status can be pending, rejected, expired, or confirmed.
func (a *API) GetDeviceFlowStatus(c *gin.Context, req *api.PollDeviceFlowRequest) (*api.DevicePollResponse, error) {
	rctx := getRequestContext(c)
	dfar, err := access.FindDeviceFlowAuthRequest(rctx, req.DeviceCode)
	if err != nil {
		return nil, err
	}

	if dfar.ExpiresAt.Before(time.Now()) {
		return &api.DevicePollResponse{
			Status:     "expired",
			DeviceCode: dfar.DeviceCode,
		}, nil
	}

	if dfar.Approved != nil && !*dfar.Approved {
		return &api.DevicePollResponse{
			Status:     "rejected",
			DeviceCode: dfar.DeviceCode,
		}, nil
	}

	if dfar.Approved != nil && *dfar.Approved {
		return &api.DevicePollResponse{
			Status:     "confirmed",
			DeviceCode: dfar.DeviceCode,
			LoginResponse: &api.LoginResponse{
				UserID:    dfar.AccessKey.IssuedFor,
				Name:      dfar.AccessKey.IssuedForName,
				AccessKey: dfar.AccessKeyToken,
				Expires:   api.Time(dfar.AccessKey.ExpiresAt),
				// TODO: set OrganizationName for consistency with other login methods
			},
		}, nil
	}

	return &api.DevicePollResponse{
		Status:     "pending",
		DeviceCode: dfar.DeviceCode,
	}, nil
}

const days = 24 * time.Hour

func (a *API) ApproveDeviceAdd(c *gin.Context, req *api.ApproveDeviceFlowRequest) (*api.EmptyResponse, error) {
	rctx := getRequestContext(c)
	dfar, err := access.FindDeviceFlowAuthRequestForApproval(rctx, strings.Replace(req.UserCode, "-", "", 1))
	if err != nil {
		return nil, err
	}

	if dfar.ExpiresAt.Before(time.Now()) {
		return nil, internal.ErrExpired
	}

	if dfar.Approved != nil && *dfar.Approved {
		// already approved, do nothing
		return nil, nil
	}

	// create access key
	user := rctx.Authenticated.User
	accessKey := &models.AccessKey{
		OrganizationMember: models.OrganizationMember{OrganizationID: rctx.Authenticated.Organization.ID},
		IssuedFor:          user.ID,
		IssuedForName:      user.Name,
		Name:               "Device " + dfar.DeviceCode,
		ExpiresAt:          rctx.Authenticated.AccessKey.ExpiresAt,
		Extension:          30 * days,
		ExtensionDeadline:  time.Now().UTC().Add(30 * days),
	}

	_, err = access.CreateAccessKey(c, accessKey)
	if err != nil {
		return nil, err
	}

	// update device flow auth request with the access key id
	err = access.SetDeviceFlowAuthRequestAccessKey(rctx, dfar.ID, accessKey)
	return nil, err
}
