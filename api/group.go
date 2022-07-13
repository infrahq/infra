package api

import (
	"github.com/infrahq/infra/internal/validate"
	"github.com/infrahq/infra/uid"
)

type Group struct {
	ID      uid.ID `json:"id"`
	Name    string `json:"name"`
	Created Time   `json:"created"`
	Updated Time   `json:"updated"`
}

type ListGroupsRequest struct {
	// Name filters the results to only the group matching this name.
	Name string `form:"name"`
	// UserID filters the results to only groups where this user is a member.
	UserID uid.ID `form:"userID"`
	PaginationRequest
}

func (r ListGroupsRequest) ValidationRules() []validate.ValidationRule {
	// no-op ValidationRules implementation so that the rules from the
	// embedded PaginationRequest struct are not applied twice.
	return nil
}

type CreateGroupRequest struct {
	Name string `json:"name"`
}

func (r CreateGroupRequest) ValidationRules() []validate.ValidationRule {
	return []validate.ValidationRule{
		ValidateName(r.Name),
		validate.Required("name", r.Name),
	}
}

type UpdateUsersInGroupRequest struct {
	GroupID         uid.ID   `uri:"id" json:"-"`
	UserIDsToAdd    []uid.ID `json:"usersToAdd"`
	UserIDsToRemove []uid.ID `json:"usersToRemove"`
}

func (r UpdateUsersInGroupRequest) ValidationRules() []validate.ValidationRule {
	return []validate.ValidationRule{
		validate.Required("id", r.GroupID),
	}
}

func (req ListGroupsRequest) GetPaginationRequest() PaginationRequest {
	return req.PaginationRequest
}

func (req ListGroupsRequest) SetPage(page int) Paginatable {

	req.PaginationRequest.Page = page

	return req
}
