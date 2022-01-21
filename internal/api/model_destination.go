package api

import (
	"encoding/json"
	"fmt"

	"github.com/infrahq/infra/uid"
)

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

// DestinationKubernetes struct for DestinationKubernetes
type DestinationKubernetes struct {
	CA       string `json:"ca" validate:"required"`
	Endpoint string `json:"endpoint" validate:"required"`
}

type ListDestinationsRequest struct {
	Kind   DestinationKind `form:"kind"`
	NodeID string          `form:"node_id"`
	Name   string          `form:"name"`
	Labels []string        `form:"labels"`
}

type CreateDestinationRequest struct {
	ID         uid.ID                 `json:"id"`
	Kind       DestinationKind        `json:"kind"`
	NodeID     string                 `json:"nodeID" validate:"required"`
	Name       string                 `json:"name" validate:"required"`
	Labels     []string               `json:"labels"`
	Kubernetes *DestinationKubernetes `json:"kubernetes,omitempty"`
}

type UpdateDestinationRequest struct {
	ID         uid.ID                 `json:"id" uri:"id" validate:"required"`
	Kind       DestinationKind        `json:"kind"`
	NodeID     string                 `json:"nodeID" validate:"required"`
	Name       string                 `json:"name" validate:"required"`
	Labels     []string               `json:"labels"`
	Kubernetes *DestinationKubernetes `json:"kubernetes,omitempty"`
}

// DestinationKind the model 'DestinationKind'
type DestinationKind string

// List of DestinationKind
const (
	DestinationKindKubernetes DestinationKind = "kubernetes"
)

var ValidDestinationKinds = []DestinationKind{
	DestinationKindKubernetes,
}

func (v *DestinationKind) UnmarshalJSON(src []byte) error {
	var value string

	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}

	enumTypeValue := DestinationKind(value)

	for _, existing := range ValidDestinationKinds {
		if existing == enumTypeValue {
			*v = enumTypeValue
			return nil
		}
	}

	return fmt.Errorf("%+v is not a valid DestinationKind", value)
}

// IsValid return true if the value is valid for the enum, false otherwise
func (v DestinationKind) IsValid() bool {
	for _, existing := range ValidDestinationKinds {
		if existing == v {
			return true
		}
	}

	return false
}
