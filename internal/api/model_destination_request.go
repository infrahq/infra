package api

import "github.com/google/uuid"

// DestinationRequest struct for DestinationRequest
type DestinationRequest struct {
	ID         uuid.UUID              `json:"id" uri:"id"`
	Kind       DestinationKind        `json:"kind"`
	NodeID     string                 `json:"nodeID"`
	Name       string                 `json:"name"`
	Labels     []string               `json:"labels"`
	Kubernetes *DestinationKubernetes `json:"kubernetes,omitempty"`
}
