package api

import (
	"github.com/infrahq/infra/internal/validate"
	"github.com/infrahq/infra/uid"
)

type Group struct {
	ID         uid.ID `json:"id" note:"Group ID" example:"gauEdoYCEU"`
	Name       string `json:"name" note:"Name of the group" example:"admins"`
	Created    Time   `json:"created" note:"Date the group was created"`
	Updated    Time   `json:"updated" note:"Date the group was updated"`
	TotalUsers int    `json:"totalUsers" note:"Total number of users in the group" example:"14"`
}

type ListGroupsRequest struct {
	// Name filters the results to only the group matching this name.
	Name string `form:"name" note:"Name of the group to retrieve" example:"admins"`
	// UserID filters the results to only groups where this user is a member.
	UserID uid.ID `form:"userID" note:"UserID of a user who is a member of the group"`
	PaginationRequest
}

func (r ListGroupsRequest) ValidationRules() []validate.ValidationRule {
	// no-op ValidationRules implementation so that the rules from the
	// embedded PaginationRequest struct are not applied twice.
	return nil
}

type CreateGroupRequest struct {
	Name string `json:"name" note:"Name of the group" example:"development"`
}

func (r CreateGroupRequest) ValidationRules() []validate.ValidationRule {
	return []validate.ValidationRule{
		validate.Required("name", r.Name),
	}
}

type UpdateUsersInGroupRequest struct {
	GroupID         uid.ID   `uri:"id" json:"-"`
	UserIDsToAdd    []uid.ID `json:"usersToAdd" note:"List of user IDs to add to the group" example:"[6dYiUyYgKa,6hPY5vqB2R]"`
	UserIDsToRemove []uid.ID `json:"usersToRemove" note:"List of  user IDs to remove from the group" example:"[3w5qrK7ets,4Ajzyzckdn]"`
}

func (r UpdateUsersInGroupRequest) ValidationRules() []validate.ValidationRule {
	return []validate.ValidationRule{
		validate.Required("id", r.GroupID),
	}
}

func (req ListGroupsRequest) SetPage(page int) Paginatable {

	req.PaginationRequest.Page = page

	return req
}
