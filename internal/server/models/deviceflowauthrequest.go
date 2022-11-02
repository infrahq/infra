package models

import (
	"time"

	"github.com/infrahq/infra/uid"
)

type DeviceFlowAuthRequest struct {
	Model
	UserCode   string
	DeviceCode string

	AccessKeyID uid.ID

	// AccessKeyToken is set once the request is approved.
	AccessKeyToken EncryptedAtRest

	ExpiresAt time.Time

	// AccessKey will be populated by some queries, but is never used on writes.
	AccessKey *AccessKey `db:"-"`
}
