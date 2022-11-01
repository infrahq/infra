package models

import (
	"time"

	"github.com/infrahq/infra/uid"
)

type DeviceFlowAuthRequest struct {
	Model
	UserCode   string
	DeviceCode string

	// TODO: remove Approved field? There's no way to reject a request right now
	// and AccessKeyID != nil indicates approved.
	Approved    *bool
	AccessKeyID uid.ID

	// AccessKeyToken is set once the request is approved.
	// TODO: use EncryptedAtRest
	AccessKeyToken string

	ExpiresAt time.Time

	// AccessKey will be populated by some queries, but is never used on writes.
	AccessKey *AccessKey `db:"-"`
}
