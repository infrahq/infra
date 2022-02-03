package api

import "github.com/infrahq/infra/uid"

// Machine struct for Machine Identities
type Machine struct {
	ID      uid.ID `json:"id"`
	Created int64  `json:"created"`
	// updated time in seconds since 1970-01-01
	Updated int64 `json:"updated"`
	// timestamp of this machine's last interaction with Infra in seconds since 1970-01-01
	LastSeenAt  int64    `json:"lastSeenAt"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Permissions []string `json:"permissions"`
}

type ListMachinesRequest struct {
	Name string `form:"name"`
}

// CreateMachineRequest struct for CreateMachineRequest
type CreateMachineRequest struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Permissions []string `json:"permissions"`
}
