package api

import (
	"github.com/infrahq/infra/uid"
)

type CreateCredentialRequest struct {
	Destination string `form:"destination"`
}

type ListCredentialRequest struct {
	Destination     string `form:"destination"`
	LastUpdateIndex int64  `form:"lastUpdateIndex"`
}

type ListCredentialRequestResponse struct {
	Items          []CredentialRequest `json:"items"`
	MaxUpdateIndex int64               `json:"maxUpdateIndex"`
}

// CredentialRequest is the database CredentialRequest object, not a request/response struct
type CredentialRequest struct {
	////////////////////
	// request fields //
	////////////////////
	ID             uid.ID `json:"id"`
	OrganizationID uid.ID `json:"organizationID"`

	UserID      uid.ID `json:"userID"`
	Destination string `json:"destination"`

	ExpiresAt Time `json:"expiresAt"`

	// Internal Fields
	UpdateIndex int64 `json:"updateIndex"`

	// certificate
	// PublicCertificate []byte `json:"publicCertificate"` // supplied if the user is planning to connect via client-generated certificate pair

	// ssh
	// PublicKey []byte  `json:"publicKey"` // supplied if the user is planning to connect via client-generated key pair

	/////////////////////
	// response fields //
	/////////////////////
	// username & pw
	// Username string
	// Password string

	// API key
	BearerToken string `json:"bearerToken"`

	// Certificate

	// JWT or generic headers
	// HeaderName string
	// Token      string
}

// UpdateCredentialRequest
type UpdateCredentialRequest struct {
	////////////////////
	// request fields //
	////////////////////
	ID             uid.ID `json:"id"`
	OrganizationID uid.ID `json:"organizationID"`
	ExpiresAt      Time   `json:"expiresAt"`

	// // certificate
	// PublicCertificate []byte // supplied if the user is planning to connect via client-generated certificate pair

	// // ssh
	// PublicKey []byte // supplied if the user is planning to connect via client-generated key pair

	// /////////////////////
	// // response fields //
	// /////////////////////
	// // username & pw
	// Username string
	// Password string

	// // API key
	BearerToken string `json:"bearerToken"`

	// // Certificate

	// // JWT or generic headers
	// HeaderName string
	// Token      string
}
