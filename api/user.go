package api

import (
	"github.com/infrahq/infra/internal/validate"
	"github.com/infrahq/infra/uid"
)

type GetUserRequest struct {
	ID IDOrSelf `uri:"id"`
}

func (r GetUserRequest) ValidationRules() []validate.ValidationRule {
	return []validate.ValidationRule{
		validate.Required("id", r.ID),
	}
}

type User struct {
	ID            uid.ID   `json:"id" note:"User ID"`
	Created       Time     `json:"created"`
	Updated       Time     `json:"updated"`
	LastSeenAt    Time     `json:"lastSeenAt"`
	Name          string   `json:"name" note:"Name of the user" example:"bob@example.com"`
	ProviderNames []string `json:"providerNames,omitempty" note:"Array of provider names" example:"['okta']"`
}

type ListUsersRequest struct {
	Name       string   `form:"name" note:"Name of the user" example:"bob@example.com"`
	Group      uid.ID   `form:"group" note:"Group the user belongs to" example:"admins"`
	IDs        []uid.ID `form:"ids" note:"Array of User IDs"`
	ShowSystem bool     `form:"showSystem" note:"if true, this shows the connector and other internal users" example:"false"`
	PaginationRequest
}

func (r ListUsersRequest) ValidationRules() []validate.ValidationRule {
	// no-op ValidationRules implementation so that the rules from the
	// embedded PaginationRequest struct are not applied twice.
	return nil
}

// CreateUserRequest is only for creating users with the Infra provider
type CreateUserRequest struct {
	Name string `json:"name"`
}

func (r CreateUserRequest) ValidationRules() []validate.ValidationRule {
	return []validate.ValidationRule{
		validate.Required("name", r.Name),
		validate.Email("name", r.Name),
	}
}

type CreateUserResponse struct {
	ID              uid.ID `json:"id"`
	Name            string `json:"name"`
	OneTimePassword string `json:"oneTimePassword,omitempty"`
}

type UpdateUserRequest struct {
	ID          uid.ID `uri:"id" json:"-"`
	OldPassword string `json:"oldPassword"`
	Password    string `json:"password"`
}

func (r UpdateUserRequest) ValidationRules() []validate.ValidationRule {
	return []validate.ValidationRule{
		validate.Required("id", r.ID),
		validate.Required("password", r.Password),
	}
}

func (req ListUsersRequest) SetPage(page int) Paginatable {
	req.PaginationRequest.Page = page

	return req
}
