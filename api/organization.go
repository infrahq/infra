package api

import (
	"github.com/infrahq/infra/internal/validate"
	"github.com/infrahq/infra/uid"
)

type Organization struct {
	ID      uid.ID `json:"id"`
	Name    string `json:"name"`
	Created Time   `json:"created"`
	Updated Time   `json:"updated"`
	Domain  string `json:"domain"`
}

type GetOrganizationRequest struct {
	ID IDOrSelf `uri:"id"`
}

type ListOrganizationsRequest struct {
	Name string `form:"name"`
	PaginationRequest
}

func (r ListOrganizationsRequest) ValidationRules() []validate.ValidationRule {
	// no-op ValidationRules implementation so that the rules from the
	// embedded PaginationRequest struct are not applied twice.
	return nil
}

type CreateOrganizationRequest struct {
	Name   string `json:"name"`
	Domain string `json:"domain"`
}

func (r CreateOrganizationRequest) ValidationRules() []validate.ValidationRule {
	return []validate.ValidationRule{
		validate.Required("name", r.Name),
		validate.Required("domain", r.Domain),
		ValidateName(r.Name),
	}
}

func (req ListOrganizationsRequest) SetPage(page int) Paginatable {
	req.PaginationRequest.Page = page
	return req
}
