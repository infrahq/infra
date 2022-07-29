package api

import "github.com/infrahq/infra/internal/validate"

type SignupEnabledResponse struct {
	Enabled bool `json:"enabled"`
}

type SignupRequest struct {
	OrgName  string `json:"orgName"`
	Name     string `json:"name"`
	Password string `json:"password"`
}

func (r SignupRequest) ValidationRules() []validate.ValidationRule {
	return []validate.ValidationRule{
		validate.Required("orgName", r.OrgName),
		validate.Required("name", r.Name),
		validate.Required("password", r.Password),
		validate.Email("name", r.Name),
	}
}
