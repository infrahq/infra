package models

import (
	"time"

	"github.com/infrahq/infra/uid"
)

// DeviceFlowAuthRequest is an outstanding request to log in via device flow
type DeviceFlowAuthRequest struct {
	Model
	UserCode   string
	DeviceCode string

	ExpiresAt time.Time

	// UserID when set means the device flow request has been approved by this user
	UserID uid.ID

	// ProviderID when set means the device flow request has been approved by a user with this provider
	ProviderID uid.ID
}

func (dr *DeviceFlowAuthRequest) Approved() bool {
	return dr.UserID != 0 && dr.ProviderID != 0
}
