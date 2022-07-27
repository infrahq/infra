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
	Name string `json:"name"`
}

func (r CreateOrganizationRequest) ValidationRules() []validate.ValidationRule {
	return []validate.ValidationRule{
		validate.Required("name", r.Name),
	}
}

func (req ListOrganizationsRequest) SetPage(page int) Paginatable {
	req.PaginationRequest.Page = page
	return req
}
