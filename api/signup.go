package api

import "github.com/infrahq/infra/internal/validate"

type SignupEnabledResponse struct {
	Enabled bool `json:"enabled"`
}

type SignupRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"` // #1825: remove, this is for migration
	Password string `json:"password"`
}

func (r SignupRequest) ValidationRules() []validate.ValidationRule {
	return []validate.ValidationRule{
		validate.Required("password", r.Password),
		validate.RequireOneOf(
			validate.Field{Name: "name", Value: r.Name},
			validate.Field{Name: "email", Value: r.Email},
		),
		validate.Email("name", r.Name),
		validate.Email("email", r.Email),
	}
}
