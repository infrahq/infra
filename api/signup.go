package api

import "github.com/infrahq/infra/internal/validate"

type SignupRequest struct {
	Name     string `json:"name"`
	Password string `json:"password"`
	Org      string `json:"org"`
}

func (r SignupRequest) ValidationRules() []validate.ValidationRule {
	return []validate.ValidationRule{
		validate.Required("name", r.Name),
		validate.Required("password", r.Password),
		validate.Required("org", r.Org),
		validate.Email("name", r.Name),
	}
}
