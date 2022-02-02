package api

import (
	"time"

	"github.com/infrahq/infra/uid"
)

// InfraAccessKey struct for InfraAccessKey
type AccessKey struct {
	ID          uid.ID    `json:"id"`
	Created     time.Time `json:"created"`
	Expires     time.Time `json:"expires,omitempty"`
	Name        string    `json:"name"`
	Permissions []string  `json:"permissions"`
}

type CreateAccessKeyRequest struct {
	Name        string        `json:"name"`
	Permissions []string      `json:"permissions"`
	Ttl         time.Duration `json:"ttl,omitempty"`
}

type CreateAccessKeyResponse struct {
	AccessKey   string    `json:"access-key"`
	ID          uid.ID    `json:"id"`
	Created     time.Time `json:"created"`
	Expires     time.Time `json:"expires,omitempty"`
	Name        string    `json:"name"`
	Permissions []string  `json:"permissions"`
}
