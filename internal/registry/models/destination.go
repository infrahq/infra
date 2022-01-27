package models

import (
	"github.com/infrahq/infra/internal/api"
	"github.com/infrahq/infra/uid"
)

type DestinationKind string

var DestinationKindKubernetes DestinationKind = "kubernetes"

type Destination struct {
	Model

	Name     string
	Kind     DestinationKind `validate:"required"`
	NodeID   string          `gorm:"uniqueIndex:,where:deleted_at is NULL" validate:"required"` // TODO: rename to UniqueID
	Endpoint string

	Labels []Label

	// Metadata []byte

	Kubernetes DestinationKubernetes
}

type DestinationKubernetes struct {
	Model

	CA string

	DestinationID uid.ID
}

func (d *Destination) ToAPI() *api.Destination {
	if d == nil {
		return nil
	}

	result := api.Destination{
		ID:      d.ID,
		Created: d.CreatedAt.Unix(),
		Updated: d.UpdatedAt.Unix(),

		Name:   d.Name,
		Kind:   api.DestinationKind(d.Kind),
		NodeID: d.NodeID,
	}

	switch d.Kind {
	case DestinationKindKubernetes:
		result.Kubernetes = &api.DestinationKubernetes{
			CA:       d.Kubernetes.CA,
			Endpoint: d.Endpoint,
		}
	}

	for _, l := range d.Labels {
		result.Labels = append(result.Labels, l.Value)
	}

	return &result
}

func (d *Destination) FromCreateAPI(from *api.CreateDestinationRequest) error {
	d.Name = from.Name
	d.NodeID = from.NodeID
	d.Kind = DestinationKind(from.Kind)

	if kubernetes := from.Kubernetes; kubernetes != nil {
		d.Endpoint = kubernetes.Endpoint
		d.Kubernetes = DestinationKubernetes{
			CA: kubernetes.CA,
		}
	}

	automaticLabels := []string{
		string(from.Kind),
	}

	for _, l := range append(from.Labels, automaticLabels...) {
		d.Labels = append(d.Labels, Label{Value: l})
	}

	return nil
}

func (d *Destination) FromUpdateAPI(from *api.UpdateDestinationRequest) error {
	d.Name = from.Name
	d.NodeID = from.NodeID
	d.Kind = DestinationKind(from.Kind)

	if kubernetes := from.Kubernetes; kubernetes != nil {
		d.Endpoint = kubernetes.Endpoint
		d.Kubernetes = DestinationKubernetes{
			CA: kubernetes.CA,
		}
	}

	automaticLabels := []string{
		string(from.Kind),
	}

	for _, l := range append(from.Labels, automaticLabels...) {
		d.Labels = append(d.Labels, Label{Value: l})
	}

	return nil
}
