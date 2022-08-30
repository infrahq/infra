package api

import "github.com/infrahq/infra/internal/validate"

type SignupResponse struct {
	User         *User         `json:"user"`
	Organization *Organization `json:"organization"`
}

type SignupOrg struct {
	Name      string `json:"name"`
	Subdomain string `json:"subDomain"`
}

func (r SignupOrg) ValidationRules() []validate.ValidationRule {
	return []validate.ValidationRule{
		validate.Required("name", r.Name),
		validate.Required("subDomain", r.Subdomain),
	}
}

type SignupRequest struct {
	Name     string    `json:"name"`
	Password string    `json:"password"`
	Org      SignupOrg `json:"org"`
}

func (r SignupRequest) ValidationRules() []validate.ValidationRule {
	return []validate.ValidationRule{
		validate.Required("name", r.Name),
		validate.Email("name", r.Name),
		validate.Required("password", r.Password),
	}
}
