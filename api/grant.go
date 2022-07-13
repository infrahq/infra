package api

import (
	"net/http"

	"github.com/infrahq/infra/internal/validate"
	"github.com/infrahq/infra/uid"
)

type Grant struct {
	ID uid.ID `json:"id"`

	Created   Time   `json:"created"`
	CreatedBy uid.ID `json:"created_by" note:"id of the user that created the grant"`
	Updated   Time   `json:"updated"`

	User      uid.ID `json:"user,omitempty"`
	Group     uid.ID `json:"group,omitempty"`
	Privilege string `json:"privilege" note:"a role or permission"`
	Resource  string `json:"resource" note:"a resource name in Infra's Universal Resource Notation"`
}

type CreateGrantResponse struct {
	*Grant     `json:",inline"`
	WasCreated bool `json:"wasCreated"`
}

func (r *CreateGrantResponse) StatusCode() int {
	if !r.WasCreated {
		return http.StatusOK
	}
	return http.StatusCreated
}

type ListGrantsRequest struct {
	User          uid.ID `form:"user"`
	Group         uid.ID `form:"group"`
	Resource      string `form:"resource" example:"production"`
	Privilege     string `form:"privilege" example:"view"`
	ShowInherited bool   `form:"showInherited" note:"if true, this field includes grants that the user inherits through groups"`
	PaginationRequest
}

func (r ListGrantsRequest) ValidationRules() []validate.ValidationRule {
	return []validate.ValidationRule{
		validate.MutuallyExclusive(
			validate.Field{Name: "user", Value: r.User},
			validate.Field{Name: "group", Value: r.Group},
		),
	}
}

type CreateGrantRequest struct {
	User      uid.ID `json:"user"`
	Group     uid.ID `json:"group"`
	Privilege string `json:"privilege" example:"view" note:"a role or permission"`
	Resource  string `json:"resource" example:"production" note:"a resource name in Infra's Universal Resource Notation"`
}

func (r CreateGrantRequest) ValidationRules() []validate.ValidationRule {
	return []validate.ValidationRule{
		validate.RequireOneOf(
			validate.Field{Name: "user", Value: r.User},
			validate.Field{Name: "group", Value: r.Group},
		),
		validate.Required("privilege", r.Privilege),
		validate.Required("resource", r.Resource),
	}
}
