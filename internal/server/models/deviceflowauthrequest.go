package models

import (
	"time"

	"github.com/infrahq/infra/uid"
)

type DeviceFlowAuthRequest struct {
	Model
	UserCode       string
	DeviceCode     string
	Approved       *bool
	AccessKeyID    uid.ID
	AccessKeyToken string // to be filled in once approved
	ExpiresAt      time.Time

	// can be preloaded if there's an AccessKeyID
	AccessKey *AccessKey
}
