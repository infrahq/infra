package api

import (
	"github.com/infrahq/infra/internal/validate"
	"github.com/infrahq/infra/uid"
)

type AccessKey struct {
	ID                uid.ID `json:"id"`
	Created           Time   `json:"created"`
	Name              string `json:"name"`
	IssuedForName     string `json:"issuedForName"`
	IssuedFor         uid.ID `json:"issuedFor"`
	ProviderID        uid.ID `json:"providerID"`
	Expires           Time   `json:"expires" note:"key is no longer valid after this time"`
	ExtensionDeadline Time   `json:"extensionDeadline" note:"key must be used within this duration to remain valid"`
}

type ListAccessKeysRequest struct {
	UserID      uid.ID `form:"user_id"`
	Name        string `form:"name"`
	ShowExpired bool   `form:"show_expired"`
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
	TTL               Duration `json:"ttl" note:"maximum time valid"`
	ExtensionDeadline Duration `json:"extensionDeadline,omitempty" note:"How long the key is active for before it needs to be renewed. The access key must be used within this amount of time to renew validity"`
}

func (r CreateAccessKeyRequest) ValidationRules() []validate.ValidationRule {
	return []validate.ValidationRule{
		ValidateName(r.Name),
		validate.Required("userID", r.UserID),
		validate.Required("ttl", r.TTL),
		validate.Required("extensionDeadline", r.ExtensionDeadline),
	}
}

type CreateAccessKeyResponse struct {
	ID                uid.ID `json:"id"`
	Created           Time   `json:"created"`
	Name              string `json:"name"`
	IssuedFor         uid.ID `json:"issuedFor"`
	ProviderID        uid.ID `json:"providerID"`
	Expires           Time   `json:"expires" note:"after this deadline the key is no longer valid"`
	ExtensionDeadline Time   `json:"extensionDeadline" note:"the key must be used by this time to remain valid"`
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
