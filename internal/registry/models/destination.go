package models

import (
	"fmt"

	"github.com/google/uuid"

	"github.com/infrahq/infra/internal/api"
)

type DestinationKind string

var DestinationKindKubernetes DestinationKind = "kubernetes"

type Destination struct {
	Model

	Name     string
	Kind     DestinationKind
	NodeID   string `gorm:"uniqueIndex:,where:deleted_at is NULL" validate:"required"`
	Endpoint string

	Labels []Label `gorm:"many2many:destination_labels"`

	Kubernetes DestinationKubernetes
}

type DestinationKubernetes struct {
	Model

	CA string

	DestinationID uuid.UUID
}

func (d *Destination) ToAPI() api.Destination {
	result := api.Destination{
		ID:      d.ID.String(),
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

	return result
}

func (d *Destination) FromAPI(from interface{}) error {
	if request, ok := from.(*api.DestinationRequest); ok {
		d.Kind = DestinationKind(request.Kind)
		d.Name = request.Name
		d.NodeID = request.NodeID

		if kubernetes, ok := request.GetKubernetesOK(); ok {
			d.Endpoint = kubernetes.Endpoint
			d.Kubernetes = DestinationKubernetes{
				CA: kubernetes.CA,
			}
		}

		automaticLabels := []string{
			string(request.Kind),
		}

		for _, l := range append(request.Labels, automaticLabels...) {
			d.Labels = append(d.Labels, Label{Value: l})
		}

		return nil
	}

	return fmt.Errorf("unknown destination model")
}

func NewDestination(id string) (*Destination, error) {
	uuid, err := uuid.Parse(id)
	if err != nil {
		return nil, err
	}

	return &Destination{
		Model: Model{
			ID: uuid,
		},
	}, nil
}
