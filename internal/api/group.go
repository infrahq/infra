package api

import "github.com/infrahq/infra/uid"

type Group struct {
	ID         uid.ID `json:"id" swaggertype:"string" example:"42MobtmDmU"`
	Name       string `json:"name" example:"Engineering"`
	Created    int64  `json:"created" example:"1646427487"`
	Updated    int64  `json:"updated" example:"1646427981"`
	ProviderID uid.ID `json:"providerID" swaggertype:"string" example:"3VGSwuC7zf"`
}

type ListGroupsRequest struct {
	Name       string `form:"name" example:"Engineering"`
	ProviderID uid.ID `form:"provider_id" swaggertype:"string" example:"3VGSwuC7zf"`
}

type CreateGroupRequest struct {
	Name       string `json:"name" validate:"required" example:"Engineering"`
	ProviderID uid.ID `json:"providerID" validate:"required" swaggertype:"string" example:"3VGSwuC7zf"`
}
