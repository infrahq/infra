package api

// Destination struct for Destination
type Destination struct {
	ID     string          `json:"id"`
	NodeID string          `json:"nodeID" form:"nodeID"`
	Name   string          `json:"name" form:"name"`
	Kind   DestinationKind `json:"kind"`
	// created time in seconds since 1970-01-01
	Created int64 `json:"created"`
	// updated time in seconds since 1970-01-01
	Updated    int64                  `json:"updated"`
	Labels     []string               `json:"labels" form:"labels"`
	Kubernetes *DestinationKubernetes `json:"kubernetes,omitempty"`
}
