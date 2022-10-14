package access

import (
	"fmt"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

func CreateDeviceFlowAuthRequest(ctx RequestContext, d *models.DeviceFlowAuthRequest) error {

	return data.CreateDeviceFlowAuthRequest(ctx.DBTxn, d)
}

// FindDeviceFlowAuthRequest is an public(unauthenticated) request, all arguments are required and must match
func FindDeviceFlowAuthRequest(ctx RequestContext, clientID, deviceCode string) (*models.DeviceFlowAuthRequest, error) {
	dfar, err := data.GetDeviceFlowAuthRequest(ctx.DBTxn, data.ByClientIDAndDeviceCode(clientID, deviceCode))
	if err != nil {
		return nil, err
	}

	if dfar.AccessKeyID > 0 {
		dfar.AccessKey, err = data.GetAccessKey(ctx.DBTxn, data.GetAccessKeysOptions{ByID: dfar.AccessKeyID})
		if err != nil {
			return nil, err
		}
	}

	return dfar, nil
}

// FindDeviceFlowAuthRequestForApproval belongs to an authenticated endpoint; it requires a logged-in user.
func FindDeviceFlowAuthRequestForApproval(ctx RequestContext, userCode string) (*models.DeviceFlowAuthRequest, error) {
	dfar, err := data.GetDeviceFlowAuthRequest(ctx.DBTxn, data.ByUserCode(userCode))
	if err != nil {
		return nil, err
	}

	return dfar, nil
}

func SetDeficeFlowAuthRequestAccessKey(ctx RequestContext, dfarID uid.ID, accessKey *models.AccessKey) error {
	if accessKey.IssuedFor != ctx.Authenticated.User.ID {
		return fmt.Errorf("%w: cannot update device flow request with access key you don't own", internal.ErrUnauthorized)
	}

	return data.SetDeficeFlowAuthRequestAccessKey(ctx.DBTxn, dfarID, accessKey)
}
