package api

import (
	"github.com/infrahq/infra/uid"
)

// Machine struct for Machine Identities
type Machine struct {
	ID          uid.ID `json:"id"`
	Created     Time   `json:"created"`
	Updated     Time   `json:"updated"`
	LastSeenAt  Time   `json:"lastSeenAt" note:"timestamp of this machine's last interaction with Infra"`
	Name        string `json:"name" validate:"max=256,min=1,required"`
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
