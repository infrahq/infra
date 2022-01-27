package api

import "github.com/infrahq/infra/uid"

// InfraAPIToken struct for InfraAPIToken
type InfraAPIToken struct {
	ID          uid.ID   `json:"id"`
	Created     int64    `json:"created"`
	Expires     *int64   `json:"expires,omitempty"`
	Name        string   `json:"name"`
	Permissions []string `json:"permissions"`
	// Token time to live before expiry in the form XhYmZs, for example 1h30m. Defaults to 12h.
	TTL *string `json:"ttl,omitempty"`
}

type ListAPITokensRequest struct {
	KeyName string `form:"name"`
}

// InfraAPITokenCreateRequest struct for InfraAPITokenCreateRequest
type InfraAPITokenCreateRequest struct {
	Name        string   `json:"name"`
	Permissions []string `json:"permissions"`
	// Token time to live before expiry in the form XhYmZs, for example 1h30m. Defaults to 12h.
	TTL *string `json:"ttl,omitempty"`
}

// InfraAPITokenCreateResponse struct for InfraAPITokenCreateResponse
type InfraAPITokenCreateResponse struct {
	Token       string   `json:"token"`
	ID          uid.ID   `json:"id"`
	Created     int64    `json:"created"`
	Expires     *int64   `json:"expires,omitempty"`
	Name        string   `json:"name"`
	Permissions []string `json:"permissions"`
	// Token time to live before expiry in the form XhYmZs, for example 1h30m. Defaults to 12h.
	TTL *string `json:"ttl,omitempty"`
}
