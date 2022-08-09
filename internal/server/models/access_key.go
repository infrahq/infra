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
	Name string `gorm:"uniqueIndex:idx_access_keys_name,where:deleted_at is NULL"`
	// IssuedFor is the ID of the user that this access key was created for
	IssuedFor         uid.ID
	IssuedForIdentity *Identity `gorm:"foreignKey:IssuedFor" db:"-"`
	ProviderID        uid.ID

	ExpiresAt         time.Time
	Extension         time.Duration // how long to increase the lifetime extension deadline by
	ExtensionDeadline time.Time

	KeyID          string `gorm:"<-;uniqueIndex:idx_access_keys_key_id,where:deleted_at is NULL"`
	Secret         string `gorm:"-" db:"-"`
	SecretChecksum []byte

	Scopes CommaSeparatedStrings // if set, scopes limit what the key can be used for
}

func (ak *AccessKey) ToAPI() *api.AccessKey {
	issuedForName := ""
	if ak.IssuedForIdentity != nil {
		issuedForName = ak.IssuedForIdentity.Name
	}

	return &api.AccessKey{
		ID:                ak.ID,
		Name:              ak.Name,
		Created:           api.Time(ak.CreatedAt),
		IssuedFor:         ak.IssuedFor,
		IssuedForName:     issuedForName,
		ProviderID:        ak.ProviderID,
		Expires:           api.Time(ak.ExpiresAt),
		ExtensionDeadline: api.Time(ak.ExtensionDeadline),
	}
}
