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

const (
	ScopePasswordReset        string = "password-reset"
	ScopeAllowCreateAccessKey string = "create-key"
)

// AccessKey is a session token presented to the Infra server as proof of authentication
type AccessKey struct {
	Model
	OrganizationMember
	Name              string
	IssuedForUser     uid.ID
	IssuedForUserName string `db:"-"`
	ProviderID        uid.ID

	ExpiresAt           time.Time     // time at which the key must expire. Extensions to the inactivity timeout do not extend this value.
	InactivityExtension time.Duration // how long to increase the inactivity timout by
	InactivityTimeout   time.Time     // time by which the key must be used or it is forced to expire early. using the key sets this to now() + inactivity extension

	KeyID          string
	Secret         string `db:"-"`
	SecretChecksum []byte

	Scopes CommaSeparatedStrings // if set, scopes limit what the key can be used for
}

func (ak *AccessKey) ToAPI() *api.AccessKey {
	return &api.AccessKey{
		ID:                ak.ID,
		Name:              ak.Name,
		Created:           api.Time(ak.CreatedAt),
		LastUsed:          api.Time(ak.UpdatedAt), // this tracks UpdatedAt which requires the InactivityTimeout to be set, otherwise it won't be updated
		IssuedFor:         ak.IssuedForUser,
		IssuedForName:     ak.IssuedForUserName,
		ProviderID:        ak.ProviderID,
		Expires:           api.Time(ak.ExpiresAt),
		InactivityTimeout: api.Time(ak.InactivityTimeout),
		Scopes:            ak.Scopes,
	}
}

// Token is only set when creating a key from CreateAccessKey
func (ak *AccessKey) Token() string {
	if len(ak.Secret) == 0 {
		return ""
	}
	return ak.KeyID + "." + ak.Secret
}
