package api

import (
	"github.com/infrahq/infra/uid"
)

type AccessKey struct {
	ID                uid.ID            `json:"id"`
	Created           Time              `json:"created"`
	Name              string            `json:"name"`
	IssuedFor         uid.PolymorphicID `json:"issuedFor"`
	Expires           Time              `json:"expires,omitempty" note:"key is no longer valid after this time"`
	ExtensionDeadline Time              `json:"extensionDeadline" note:"key must be renewed after this time"`
}

type ListAccessKeysRequest struct {
	MachineID uid.ID `form:"machine_id"`
	Name      string `form:"name"`
}

type CreateAccessKeyRequest struct {
	MachineID         uid.ID   `json:"machineID" validate:"required"`
	Name              string   `json:"name"`
	TTL               Duration `json:"ttl" note:"maximum time valid"`
	ExtensionDeadline Duration `json:"extensionDeadline,omitempty" note:"How long the key is active for before it needs to be renewed. The access key must be used within this amount of time to renew validity"`
}

type CreateAccessKeyResponse struct {
	ID                uid.ID            `json:"id"`
	Created           Time              `json:"created"`
	Name              string            `json:"name"`
	IssuedFor         uid.PolymorphicID `json:"issuedFor"`
	Expires           Time              `json:"expires" note:"after this deadline the key is no longer valid"`
	ExtensionDeadline Time              `json:"extensionDeadline" note:"the key must be used by this time to remain valid"`
	AccessKey         string            `json:"accessKey"`
}
