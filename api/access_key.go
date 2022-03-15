package api

import (
	"time"

	"github.com/infrahq/infra/uid"
)

type AccessKey struct {
	ID                uid.ID            `json:"id"`
	Created           time.Time         `json:"created"`
	Name              string            `json:"name"`
	IssuedFor         uid.PolymorphicID `json:"issuedFor"`
	Expires           time.Time         `json:"expires,omitempty"`
	ExtensionDeadline time.Time         `json:"extensionDeadline"`
}

type ListAccessKeysRequest struct {
	MachineID uid.ID `form:"machine_id"`
	Name      string `form:"name"`
}

type CreateAccessKeyRequest struct {
	MachineID         uid.ID `json:"machineID" validate:"required"`
	Name              string `json:"name"`
	TTL               string `json:"ttl"`                         // maximum time valid
	ExtensionDeadline string `json:"extensionDeadline,omitempty"` // the access key must be used within this amount of time to renew validity
}

type CreateAccessKeyResponse struct {
	ID                uid.ID            `json:"id"`
	Created           time.Time         `json:"created"`
	Name              string            `json:"name"`
	IssuedFor         uid.PolymorphicID `json:"issuedFor"`
	Expires           time.Time         `json:"expires"`           // after this deadline the key is no longer valid
	ExtensionDeadline time.Time         `json:"extensionDeadline"` // the key must be used by this time to remain valid
	AccessKey         string            `json:"accessKey"`
}
