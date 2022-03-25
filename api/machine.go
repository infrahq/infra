package api

import (
	"github.com/infrahq/infra/uid"
)

// Machine struct for Machine Identities
type Machine struct {
	ID      uid.ID `json:"id"`
	Created Time   `json:"created"`
	// updated time in seconds since 1970-01-01
	Updated Time `json:"updated"`
	// timestamp of this machine's last interaction with Infra in seconds since 1970-01-01
	LastSeenAt  Time   `json:"lastSeenAt"`
	Name        string `json:"name" validate:"max=256,required"`
	Description string `json:"description"`
}

type ListMachinesRequest struct {
	Name string `form:"name"`
}

// CreateMachineRequest struct for CreateMachineRequest
type CreateMachineRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}
