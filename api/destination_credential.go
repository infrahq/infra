package api

import (
	"github.com/infrahq/infra/uid"
)

type CreateDestinationCredentialRequest struct {
	Destination string `form:"destination"`
}

type ListDestinationCredentialRequest struct {
	Destination     string `form:"destination"`
	LastUpdateIndex int64  `form:"lastUpdateIndex"`
}

type ListDestinationCredentialResponse struct {
	Items          []DestinationCredential `json:"items"`
	MaxUpdateIndex int64                   `json:"maxUpdateIndex"`
}

// DestinationCredential represents the request for credentials to a particular destination
type DestinationCredential struct {
	////////////////////
	// request fields //
	////////////////////
	ID             uid.ID `json:"id"`
	OrganizationID uid.ID `json:"organizationID"`

	UserID      uid.ID `json:"userID"`
	Destination string `json:"destination"`

	RequestExpiresAt Time `json:"requestExpiresAt"`

	// Internal Fields
	UpdateIndex int64 `json:"updateIndex"`

	/////////////////////
	// response fields //
	/////////////////////

	// API key
	BearerToken         string `json:"bearerToken"`
	CredentialExpiresAt Time   `json:"credentialExpiresAt"`

	// Certificate

	// JWT or generic headers
	// HeaderName string
	// Token      string
}

// AnswerDestinationCredentialRequest
type AnswerDestinationCredentialRequest struct {
	ID             uid.ID `json:"id"`
	OrganizationID uid.ID `json:"organizationID"`

	// API key
	BearerToken         string `json:"bearerToken"`
	CredentialExpiresAt Time   `json:"credentialExpiresAt"`
}
