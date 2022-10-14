package server

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/access"
	"github.com/infrahq/infra/internal/generate"
	"github.com/infrahq/infra/internal/logging"
	"github.com/infrahq/infra/internal/server/models"
)

const DeviceCodeExpirySeconds = 1800

func (a *API) StartDeviceFlow(c *gin.Context, req *api.StartDeviceFlowRequest) (*api.DeviceFlowResponse, error) {
	rctx := getRequestContext(c)
	userCode, err := generate.CryptoRandom(8, generate.CharsetDeviceCode)
	if err != nil {
		return nil, err
	}
	userCode = userCode[0:4] + "-" + userCode[4:]

	deviceCode, err := generate.CryptoRandom(38, generate.CharsetAlphaNumeric)
	if err != nil {
		return nil, err
	}

	err = access.CreateDeviceFlowAuthRequest(rctx, &models.DeviceFlowAuthRequest{
		ClientID:   req.ClientID,
		UserCode:   userCode,
		DeviceCode: deviceCode,
		ExpiresAt:  time.Now().Add(DeviceCodeExpirySeconds * time.Second),
	})
	if err != nil {
		return nil, err
	}
	host := a.server.options.BaseDomain
	if rctx.Authenticated.Organization != nil {
		host = rctx.Authenticated.Organization.Domain
	}

	return &api.DeviceFlowResponse{
		DeviceCode:          deviceCode,
		VerificationURI:     fmt.Sprintf("https://%s/device", host),
		UserCode:            userCode,
		ExpiresInSeconds:    DeviceCodeExpirySeconds,
		PollIntervalSeconds: 5,
	}, nil
}

// can error with one of authorization_pending, access_denied, expired_token, slow_down
func (a *API) GetDeviceFlowStatus(c *gin.Context, req *api.PollDeviceFlowRequest) (*api.DevicePollResponse, error) {
	rctx := getRequestContext(c)
	dfar, err := access.FindDeviceFlowAuthRequest(rctx, req.ClientID, req.DeviceCode)
	if err != nil {
		return nil, err
	}
	logging.Debugf("Found record")

	if dfar.ExpiresAt.Before(time.Now()) {
		logging.Debugf("it's expired")
		return &api.DevicePollResponse{
			Status:     "expired",
			DeviceCode: dfar.DeviceCode,
		}, nil
	}

	if dfar.Approved != nil && !*dfar.Approved {
		logging.Debugf("it was rejected")
		return &api.DevicePollResponse{
			Status:     "rejected",
			DeviceCode: dfar.DeviceCode,
		}, nil
	}

	if dfar.Approved != nil && *dfar.Approved {
		logging.Debugf("it's approved")
		return &api.DevicePollResponse{
			Status:     "confirmed",
			DeviceCode: dfar.DeviceCode,
			LoginResponse: &api.LoginResponse{
				UserID:    dfar.AccessKey.IssuedFor,
				Name:      dfar.AccessKey.IssuedForName,
				AccessKey: dfar.AccessKeyToken,
				Expires:   api.Time(dfar.AccessKey.ExpiresAt),
			},
		}, nil
	}

	logging.Debugf("it's pending")

	return &api.DevicePollResponse{
		Status:     "pending",
		DeviceCode: dfar.DeviceCode,
	}, nil
}

const (
	day  = 24 * time.Hour
	year = 365 * day
)

func (a *API) ApproveDeviceAdd(c *gin.Context, req *api.ApproveDeviceFlowRequest) (*api.EmptyResponse, error) {
	rctx := getRequestContext(c)
	dfar, err := access.FindDeviceFlowAuthRequestForApproval(rctx, req.UserCode)
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
		Name:               "Device " + dfar.ClientID + ":" + dfar.DeviceCode,
		ExpiresAt:          time.Now().UTC().Add(time.Duration(10 * year)),
		Extension:          time.Duration(30 * day),
		ExtensionDeadline:  time.Now().UTC().Add(30 * day),
	}

	_, err = access.CreateAccessKey(c, accessKey)
	if err != nil {
		return nil, err
	}

	// update device flow auth request with the access key id
	err = access.SetDeficeFlowAuthRequestAccessKey(rctx, dfar.ID, accessKey)
	if err != nil {
		return nil, err
	}

	// success
	return nil, nil
}
