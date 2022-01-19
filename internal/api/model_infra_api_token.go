package api

// InfraAPIToken struct for InfraAPIToken
type InfraAPIToken struct {
	ID          string   `json:"id"`
	Created     int64    `json:"created"`
	Expires     *int64   `json:"expires,omitempty"`
	Name        string   `json:"name"`
	Permissions []string `json:"permissions"`
	// Token time to live before expiry in the form XhYmZs, for example 1h30m. Defaults to 12h.
	Ttl *string `json:"ttl,omitempty"`
}

type ListAPITokensRequest struct {
	KeyName string `form:"name"`
}
