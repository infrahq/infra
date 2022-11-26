package access

import (
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
)

func CreateDeviceFlowAuthRequest(ctx RequestContext, d *models.DeviceFlowAuthRequest) error {
	return data.CreateDeviceFlowAuthRequest(ctx.DBTxn, d)
}

// FindDeviceFlowAuthRequest is an public(unauthenticated) request, all arguments are required and must match
func FindDeviceFlowAuthRequest(ctx RequestContext, deviceCode string) (*models.DeviceFlowAuthRequest, error) {
	dfar, err := data.GetDeviceFlowAuthRequest(ctx.DBTxn, data.GetDeviceFlowAuthRequestOptions{ByDeviceCode: deviceCode})
	if err != nil {
		return nil, err
	}

	return dfar, nil
}

// FindDeviceFlowAuthRequestForApproval belongs to an authenticated endpoint; it requires a logged-in user.
func FindDeviceFlowAuthRequestForApproval(ctx RequestContext, userCode string) (*models.DeviceFlowAuthRequest, error) {
	return data.GetDeviceFlowAuthRequest(ctx.DBTxn, data.GetDeviceFlowAuthRequestOptions{ByUserCode: userCode})
}
