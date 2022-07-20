package models

import (
	"time"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/uid"
)

var (
	AccessKeyKeyLength    = 10 // the length of the ID used to look-up the access key
	AccessKeySecretLength = 24 // the length of the secret used to validate an access key
)

const ScopePasswordReset = "password-reset"

// AccessKey is a session token presented to the Infra server as proof of authentication
type AccessKey struct {
	Model
	Name          string                `gorm:"uniqueIndex:idx_access_keys_name,where:deleted_at is NULL" validate:"excludes= "`
	IssuedFor     uid.ID                `validate:"required"` // the ID of the identity that this access key was created for
	IssuedForName string                `gorm:"-"`
	ProviderID    uid.ID                `validate:"required"`
	Scopes        CommaSeparatedStrings // if set, scopes limit what the key can be used for

	ExpiresAt         time.Time     `validate:"required"`
	Extension         time.Duration // how long to increase the lifetime extension deadline by
	ExtensionDeadline time.Time

	KeyID          string `gorm:"<-;uniqueIndex:idx_access_keys_key_id,where:deleted_at is NULL"`
	Secret         string `gorm:"-"`
	SecretChecksum []byte
}

func (ak *AccessKey) ToAPI() *api.AccessKey {
	return &api.AccessKey{
		ID:                ak.ID,
		Name:              ak.Name,
		Created:           api.Time(ak.CreatedAt),
		IssuedFor:         ak.IssuedFor,
		IssuedForName:     ak.IssuedForName,
		ProviderID:        ak.ProviderID,
		Expires:           api.Time(ak.ExpiresAt),
		ExtensionDeadline: api.Time(ak.ExtensionDeadline),
	}
}
