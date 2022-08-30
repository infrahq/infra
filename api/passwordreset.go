package api

import "github.com/infrahq/infra/internal/validate"

type PasswordResetRequest struct {
	Email string `json:"email"`
}

func (r PasswordResetRequest) ValidationRules() []validate.ValidationRule {
	return []validate.ValidationRule{
		validate.Required("email", r.Email),
		validate.Email("email", r.Email),
	}
}

type VerifiedResetPasswordRequest struct {
	Token    string `json:"token"`
	Password string `json:"password"`
}

func (r VerifiedResetPasswordRequest) ValidationRules() []validate.ValidationRule {
	return []validate.ValidationRule{
		validate.Required("token", r.Token),
		validate.StringRule{
			Name:            "token",
			Value:           r.Token,
			MinLength:       10,
			MaxLength:       10,
			CharacterRanges: validate.AlphaNumeric,
		},
		validate.Required("password", r.Password),
	}
}
