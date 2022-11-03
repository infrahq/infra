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
	Name string
	/* IssuedFor is either:
	1. The ID of the user that this access key was created for.
	2. The ID of a provider that is doing SCIM provisioning using this access key.
	*/
	IssuedFor     uid.ID
	IssuedForName string `db:"-"`
	ProviderID    uid.ID

	ExpiresAt         time.Time     // time at which the key must expire. Extensions do not extend this value.
	Extension         time.Duration // how long to increase the lifetime extension deadline by
	ExtensionDeadline time.Time     // time by which the key must be used or it is forced to expire early. using the key sets this to now() + Extension

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
		IssuedFor:         ak.IssuedFor,
		IssuedForName:     ak.IssuedForName,
		ProviderID:        ak.ProviderID,
		Expires:           api.Time(ak.ExpiresAt),
		ExtensionDeadline: api.Time(ak.ExtensionDeadline),
	}
}

// Token is only set when creating a key from CreateAccessKey
func (ak *AccessKey) Token() string {
	if len(ak.Secret) == 0 {
		return ""
	}
	return ak.KeyID + "." + ak.Secret
}
