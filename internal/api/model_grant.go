package api

// Grant struct for Grant
type Grant struct {
	ID string `json:"id"`
	// created time in seconds since 1970-01-01
	Created int64 `json:"created"`
	// updated time in seconds since 1970-01-01
	Updated     int64            `json:"updated"`
	Kind        GrantKind        `json:"kind"`
	Destination *Destination     `json:"destination"`
	Kubernetes  *GrantKubernetes `json:"kubernetes,omitempty"`
	Users       []User           `json:"users,omitempty"`
	Groups      []Group          `json:"groups,omitempty"`
}
