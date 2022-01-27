package api

import "github.com/infrahq/infra/uid"

// Group struct for Group
type Group struct {
	ID   uid.ID `json:"id"`
	Name string `json:"name"`
	// created time in seconds since 1970-01-01
	Created int64 `json:"created"`
	// updated time in seconds since 1970-01-01
	Updated   int64      `json:"updated"`
	Users     []User     `json:"users,omitempty"`
	Grants    []Grant    `json:"grants,omitempty"`
	Providers []Provider `json:"providers,omitempty"`
}

type ListGroupsRequest struct {
	GroupName string `form:"name"`
}
