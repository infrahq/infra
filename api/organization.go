package api

import (
	"github.com/gin-gonic/gin"
	"github.com/infrahq/infra/internal/validate"
	"github.com/infrahq/infra/uid"
)

type Organization struct {
	ID             uid.ID   `json:"id"`
	Name           string   `json:"name"`
	Created        Time     `json:"created"`
	Updated        Time     `json:"updated"`
	Domain         string   `json:"domain"`
	AllowedDomains []string `json:"allowedDomains" note:"domains which can be used to login to this organization" example:"['example.com', 'infrahq.com']"`
}

type GetOrganizationRequest struct {
	ID IDOrSelf `uri:"id"`
}

func (r *GetOrganizationRequest) SetFromParams(p gin.Params) error {
	return r.ID.SetFromParams(p)
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

type UpdateOrganizationRequest struct {
	Resource
	AllowedDomains []string `json:"allowedDomains"`
}

func (r UpdateOrganizationRequest) ValidationRules() []validate.ValidationRule {
	return []validate.ValidationRule{
		validate.Required("id", r.ID),
		validate.Required("allowedDomains", r.AllowedDomains),
		// permissive validation for a domain field (with no protocol)
		validate.SliceRule{
			Value: r.AllowedDomains,
			Name:  "allowedDomains",
			ItemRule: validate.StringRule{
				Name:      "allowedDomains.values",
				MinLength: 2,
				MaxLength: 254,
				CharacterRanges: []validate.CharRange{
					validate.AlphabetLower,
					validate.AlphabetUpper,
					validate.Numbers,
					validate.Dash,
					validate.Dot,
				},
				FirstCharacterRange: validate.AlphaNumeric,
				RequiredCharacters:  []rune{'.'},
				DenyList:            []string{"gmail.com", "googlemail.com"},
			},
		},
	}
}

func (req ListOrganizationsRequest) SetPage(page int) Paginatable {
	req.PaginationRequest.Page = page
	return req
}
