package api

import "github.com/infrahq/infra/uid"

type Group struct {
	ID         uid.ID `json:"id"`
	Name       string `json:"name"`
	Created    int64  `json:"created"`
	Updated    int64  `json:"updated"`
	ProviderID uid.ID `json:"providerID"`
}

type ListGroupsRequest struct {
	Name       string `form:"name"`
	ProviderID uid.ID `form:"provider_id"`
}

type CreateGroupRequest struct {
	Name       string `json:"name" validate:"required"`
	ProviderID uid.ID `json:"providerID" validate:"required"`
}
