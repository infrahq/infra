package api

// Group struct for Group
type Group struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	// created time in seconds since 1970-01-01
	Created int64 `json:"created"`
	// updated time in seconds since 1970-01-01
	Updated   int64      `json:"updated"`
	Users     []User     `json:"users,omitempty"`
	Grants    []Grant    `json:"grants,omitempty"`
	Providers []Provider `json:"providers,omitempty"`
}
