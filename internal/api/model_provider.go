package api

import (
	"encoding/json"
	"fmt"

	"github.com/infrahq/infra/uuid"
)

// Provider struct for Provider
type Provider struct {
	ID string `json:"id"`
	// created time in seconds since 1970-01-01
	Created int64 `json:"created"`
	// updated time in seconds since 1970-01-01
	Updated  int64        `json:"updated"`
	Domain   string       `json:"domain" validate:"fqdn,required"`
	ClientID string       `json:"clientID" validate:"required"`
	Kind     ProviderKind `json:"kind"`
}

type CreateProviderRequest struct {
	Kind         ProviderKind  `json:"kind" validate:"required"`
	Domain       string        `json:"domain" validate:"fqdn,required"`
	ClientID     string        `json:"clientID"`
	ClientSecret string        `json:"clientSecret"`
	Okta         *ProviderOkta `json:"okta,omitempty"`
}

type UpdateProviderRequest struct {
	ID           uuid.UUID     `uri:"id" json:"id"`
	Kind         ProviderKind  `json:"kind"`
	Domain       string        `json:"domain" validate:"fqdn,required"`
	ClientID     string        `json:"clientID"`
	ClientSecret string        `json:"clientSecret"`
	Okta         *ProviderOkta `json:"okta,omitempty"`
}

type ListProvidersRequest struct {
	ProviderKind ProviderKind `form:"kind"`
	Domain       string       `form:"domain"`
}

type ProviderOkta struct {
	APIToken string `json:"apiToken"`
}

// ProviderKind the model 'ProviderKind'
type ProviderKind string

// List of ProviderKind
const (
	ProviderKindOkta ProviderKind = "okta"
)

var ValidProviderKinds = []ProviderKind{
	ProviderKindOkta,
}

func (v *ProviderKind) UnmarshalJSON(src []byte) error {
	var value string

	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}

	enumTypeValue := ProviderKind(value)

	for _, existing := range ValidProviderKinds {
		if existing == enumTypeValue {
			*v = enumTypeValue
			return nil
		}
	}

	return fmt.Errorf("%+v is not a valid ProviderKind", value)
}
