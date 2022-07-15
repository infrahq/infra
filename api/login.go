package api

import (
	"github.com/infrahq/infra/internal/validate"
	"github.com/infrahq/infra/uid"
)

type LoginRequestOIDC struct {
	ProviderID  uid.ID `json:"providerID"`
	RedirectURL string `json:"redirectURL"`
	Code        string `json:"code"`
}

func (r LoginRequestOIDC) ValidationRules() []validate.ValidationRule {
	return []validate.ValidationRule{
		validate.Required("providerID", r.ProviderID),
		validate.Required("redirectURL", r.RedirectURL),
		validate.Required("code", r.Code),
	}
}

type LoginRequestPasswordCredentials struct {
	Name     string `json:"name"`
	Email    string `json:"email"` // #1825: remove, this is for migration
	Password string `json:"password"`
}

func (r LoginRequestPasswordCredentials) ValidationRules() []validate.ValidationRule {
	return []validate.ValidationRule{
		validate.RequireOneOf(
			validate.Field{Name: "name", Value: r.Name},
			validate.Field{Name: "email", Value: r.Email},
		),
		validate.Required("password", r.Password),
	}
}

type LoginRequest struct {
	AccessKey           string                           `json:"accessKey"`
	PasswordCredentials *LoginRequestPasswordCredentials `json:"passwordCredentials"`
	OIDC                *LoginRequestOIDC                `json:"oidc"`
}

func (r LoginRequest) ValidationRules() []validate.ValidationRule {
	return []validate.ValidationRule{
		validate.RequireOneOf(
			validate.Field{Name: "accessKey", Value: r.AccessKey},
			validate.Field{Name: "passwordCredentials", Value: r.PasswordCredentials},
			validate.Field{Name: "oidc", Value: r.OIDC},
		),
	}
}

type LoginResponse struct {
	UserID                 uid.ID `json:"userID"`
	Name                   string `json:"name"`
	AccessKey              string `json:"accessKey"`
	PasswordUpdateRequired bool   `json:"passwordUpdateRequired,omitempty"`
	Expires                Time   `json:"expires"`
}
