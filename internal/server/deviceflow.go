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

	err = data.CreateDeviceFlowAuthRequest(rctx.DBTxn, &models.DeviceFlowAuthRequest{
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

	var host string
	if rctx.Authenticated.Organization != nil {
		host = rctx.Authenticated.Organization.Domain
	}

	if host == "" {
		host = rctx.Request.Host
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
// flow login. The response status can be pending, expired, or confirmed.
func (a *API) GetDeviceFlowStatus(c *gin.Context, req *api.DeviceFlowStatusRequest) (*api.DeviceFlowStatusResponse, error) {
	rctx := getRequestContext(c)

	dfar, err := data.GetDeviceFlowAuthRequest(rctx.DBTxn, data.GetDeviceFlowAuthRequestOptions{ByDeviceCode: req.DeviceCode})
	if err != nil {
		return nil, fmt.Errorf("%w: error retrieving device flow auth request: %v", internal.ErrUnauthorized, err)
	}

	if dfar.ExpiresAt.Before(time.Now()) {
		return &api.DeviceFlowStatusResponse{
			Status:     api.DeviceFlowStatusExpired,
			DeviceCode: dfar.DeviceCode,
		}, nil
	}

	if !dfar.Approved() {
		return &api.DeviceFlowStatusResponse{
			Status:     api.DeviceFlowStatusPending,
			DeviceCode: dfar.DeviceCode,
		}, nil
	}

	user, err := data.GetIdentity(rctx.DBTxn, data.GetIdentityOptions{ByID: dfar.UserID})
	if err != nil {
		return nil, fmt.Errorf("%w: retrieving approval user: %v", internal.ErrUnauthorized, err)
	}

	accessKey := &models.AccessKey{
		IssuedFor:     user.ID,
		IssuedForName: user.Name,

		// Share the same provider ID that was used to approve
		ProviderID:          dfar.ProviderID,
		ExpiresAt:           time.Now().UTC().Add(a.server.options.SessionDuration),
		InactivityTimeout:   time.Now().UTC().Add(a.server.options.SessionInactivityTimeout),
		InactivityExtension: a.server.options.SessionInactivityTimeout,
		Scopes:              models.CommaSeparatedStrings{models.ScopeAllowCreateAccessKey},
	}

	bearer, err := data.CreateAccessKey(rctx.DBTxn, accessKey)
	if err != nil {
		return nil, fmt.Errorf("%w: creating new access key: %v", internal.ErrUnauthorized, err)
	}

	user.LastSeenAt = time.Now().UTC()
	if err := data.UpdateIdentity(rctx.DBTxn, user); err != nil {
		return nil, fmt.Errorf("%w: update user last seen: %v", internal.ErrUnauthorized, err)
	}

	a.t.User(accessKey.IssuedFor.String(), user.Name)
	a.t.OrgMembership(accessKey.OrganizationID.String(), accessKey.IssuedFor.String())
	a.t.Event("login", accessKey.IssuedFor.String(), accessKey.OrganizationID.String(), Properties{"method": "deviceflow"})

	// Update the request context so that logging middleware can include the userID
	rctx.Authenticated.User = user
	c.Set(access.RequestContextKey, rctx)

	org, err := data.GetOrganization(rctx.DBTxn, data.GetOrganizationOptions{ByID: accessKey.OrganizationID})
	if err != nil {
		return nil, fmt.Errorf("%w: device flow get organization for user: %v", internal.ErrUnauthorized, err)
	}

	// Delete the request so it can't be claimed twice
	err = data.DeleteDeviceFlowAuthRequest(rctx.DBTxn, dfar.ID)
	if err != nil {
		return nil, fmt.Errorf("%w: device flow delete auth request: %v", internal.ErrUnauthorized, err)
	}

	return &api.DeviceFlowStatusResponse{
		Status:     api.DeviceFlowStatusConfirmed,
		DeviceCode: dfar.DeviceCode,
		LoginResponse: &api.LoginResponse{
			UserID:           accessKey.IssuedFor,
			Name:             accessKey.IssuedForName,
			AccessKey:        string(bearer),
			Expires:          api.Time(accessKey.ExpiresAt),
			OrganizationName: org.Name,
		},
	}, nil
}

func (a *API) ApproveDeviceFlow(c *gin.Context, req *api.ApproveDeviceFlowRequest) (*api.EmptyResponse, error) {
	rctx := getRequestContext(c)

	if !rctx.Authenticated.AccessKey.Scopes.Includes(models.ScopeAllowCreateAccessKey) {
		return nil, fmt.Errorf("%w: access key missing scope '%s'", access.ErrNotAuthorized, models.ScopeAllowCreateAccessKey)
	}

	dfar, err := data.GetDeviceFlowAuthRequest(rctx.DBTxn, data.GetDeviceFlowAuthRequestOptions{ByUserCode: strings.Replace(req.UserCode, "-", "", 1)})
	if err != nil {
		return nil, fmt.Errorf("%w: invalid code", internal.ErrNotFound)
	}

	if dfar.ExpiresAt.Before(time.Now()) {
		return nil, internal.ErrExpired
	}

	if dfar.Approved() {
		return nil, nil
	}

	return nil, data.ApproveDeviceFlowAuthRequest(rctx.DBTxn, dfar.ID, rctx.Authenticated.User.ID, rctx.Authenticated.AccessKey.ProviderID)
}
