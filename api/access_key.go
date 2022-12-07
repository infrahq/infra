package api

import (
	"github.com/infrahq/infra/internal/validate"
	"github.com/infrahq/infra/uid"
)

type AccessKey struct {
	ID                uid.ID   `json:"id" note:"ID of the access key"`
	Created           Time     `json:"created"`
	LastUsed          Time     `json:"lastUsed"`
	Name              string   `json:"name" example:"cicdkey" note:"Name of the access key"`
	IssuedForName     string   `json:"issuedForName" example:"admin@example.com" note:"Name of the user the key was issued to"`
	IssuedFor         uid.ID   `json:"issuedFor" note:"ID of the user the key was issued to"`
	ProviderID        uid.ID   `json:"providerID" note:"ID of the provider if the user is managed by an OIDC provider"`
	Expires           Time     `json:"expires" note:"key is no longer valid after this time"`
	InactivityTimeout Time     `json:"inactivityTimeout" note:"key must be used by this time to remain valid"`
	Scopes            []string `json:"scopes" note:"additional access level scopes that control what an access key can do"`
}

type ListAccessKeysRequest struct {
	UserID      uid.ID `form:"userID" note:"UserID of the user whose access keys you want to list"`
	Name        string `form:"name" note:"Name of the user" example:"john@example.com"`
	ShowExpired bool   `form:"showExpired" note:"Whether to show expired access keys. Defaults to false" example:"true"`
	PaginationRequest
}

func (r ListAccessKeysRequest) ValidationRules() []validate.ValidationRule {
	// no-op ValidationRules implementation so that the rules from the
	// embedded PaginationRequest struct are not applied twice.
	return nil
}

type CreateAccessKeyRequest struct {
	UserID            uid.ID   `json:"userID"`
	Name              string   `json:"name"`
	Expiry            Duration `json:"expiry" note:"maximum time valid"`
	InactivityTimeout Duration `json:"inactivityTimeout" note:"key must be used within this duration to remain valid"`
}

func (r CreateAccessKeyRequest) ValidationRules() []validate.ValidationRule {
	return []validate.ValidationRule{
		ValidateName(r.Name),
		validate.Required("userID", r.UserID),
		validate.Required("expiry", r.Expiry),
		validate.Required("inactivityTimeout", r.InactivityTimeout),
	}
}

type CreateAccessKeyResponse struct {
	ID                uid.ID `json:"id"`
	Created           Time   `json:"created"`
	Name              string `json:"name"`
	IssuedFor         uid.ID `json:"issuedFor"`
	ProviderID        uid.ID `json:"providerID"`
	Expires           Time   `json:"expires" note:"after this deadline the key is no longer valid"`
	InactivityTimeout Time   `json:"inactivityTimeout" note:"the key must be used by this time to remain valid"`
	AccessKey         string `json:"accessKey"`
}

// ValidateName returns a standard validation rule for all name fields. The
// field name must always be "name".
func ValidateName(value string) validate.StringRule {
	return validate.StringRule{
		Value:     value,
		Name:      "name",
		MinLength: 2,
		MaxLength: 256,
		CharacterRanges: []validate.CharRange{
			validate.AlphabetLower,
			validate.AlphabetUpper,
			validate.Numbers,
			validate.Dash, validate.Underscore, validate.Dot,
		},
	}
}

func (req ListAccessKeysRequest) SetPage(page int) Paginatable {
	req.PaginationRequest.Page = page

	return req
}

type DeleteAccessKeyRequest struct {
	Name string `form:"name" note:"Name of the access key to delete" example:"cicdkey"`
}

func (r DeleteAccessKeyRequest) ValidationRules() []validate.ValidationRule {
	return []validate.ValidationRule{
		ValidateName(r.Name),
		validate.Required("name", r.Name),
	}
}
