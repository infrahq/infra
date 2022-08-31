package api

import "github.com/infrahq/infra/internal/validate"

type ForgotDomainRequest struct {
	Email string `json:"email"`
}

func (r ForgotDomainRequest) ValidationRules() []validate.ValidationRule {
	return []validate.ValidationRule{
		validate.Required("email", r.Email),
		validate.Email("email", r.Email),
	}
}
