package api

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
