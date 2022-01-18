package api

import (
	"time"

	"github.com/infrahq/infra/uid"
)

// InfraAPIToken struct for InfraAPIToken
type APIToken struct {
	ID          uid.ID    `json:"id"`
	Created     time.Time `json:"created"`
	Expires     time.Time `json:"expires,omitempty"`
	Name        string    `json:"name"`
	Permissions []string  `json:"permissions"`
}

type CreateAPITokenRequest struct {
	Name        string        `json:"name"`
	Permissions []string      `json:"permissions"`
	Ttl         time.Duration `json:"ttl,omitempty"`
}

type CreateAPITokenResponse struct {
	Token       string    `json:"token"`
	ID          uid.ID    `json:"id"`
	Created     time.Time `json:"created"`
	Expires     time.Time `json:"expires,omitempty"`
	Name        string    `json:"name"`
	Permissions []string  `json:"permissions"`
}
