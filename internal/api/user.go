package api

import "github.com/infrahq/infra/uid"

type User struct {
	ID         uid.ID `json:"id" swaggertype:"string" example:"42MobtmDmU"`
	Email      string `json:"email" example:"example@infrahq.com"`
	Created    int64  `json:"created" example:"1646427487"`
	Updated    int64  `json:"updated" example:"1646427981"`
	LastSeenAt int64  `json:"lastSeenAt" example:"1646427487"`
	ProviderID uid.ID `json:"providerID" swaggertype:"string" example:"3VGSwuC7zf"`
}

type ListUsersRequest struct {
	Email      string `form:"email" example:"example@infrahq.com"`
	ProviderID uid.ID `form:"provider_id" swaggertype:"string" example:"3VGSwuC7zf"`
}

type CreateUserRequest struct {
	Email      string `json:"email" validate:"email,required" example:"example@infrahq.com"`
	ProviderID uid.ID `json:"providerID" validate:"required" swaggertype:"string" example:"3VGSwuC7zf"`
}
