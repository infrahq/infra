package models

import (
	"github.com/google/uuid"

	"github.com/infrahq/infra/internal/api"
)

type DestinationKind string

var DestinationKindKubernetes DestinationKind = "kubernetes"

type Destination struct {
	Model

	Name     string
	Kind     DestinationKind
	NodeID   string
	Endpoint string

	Labels []Label

	Kubernetes DestinationKubernetes
}

type DestinationKubernetes struct {
	Model

	CA string

	DestinationID uuid.UUID
}

func (d *Destination) ToAPI() api.Destination {
	result := api.Destination{
		Id:      d.ID.String(),
		Created: d.CreatedAt.Unix(),
		Updated: d.UpdatedAt.Unix(),

		Name:   d.Name,
		Kind:   api.DestinationKind(d.Kind),
		NodeID: d.NodeID,
	}

	switch d.Kind {
	case DestinationKindKubernetes:
		result.Kubernetes = &api.DestinationKubernetes{
			Ca:       d.Kubernetes.CA,
			Endpoint: d.Endpoint,
		}
	}

	for _, l := range d.Labels {
		result.Labels = append(result.Labels, l.Value)
	}

	return result
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

func (d *Destination) FromAPICreateRequest(template *api.DestinationCreateRequest) error {
	d.Name = template.Name
	d.NodeID = template.NodeID
	d.Kind = DestinationKind(template.Kind)
	d.Endpoint = template.Kubernetes.Endpoint

	switch d.Kind {
	case DestinationKindKubernetes:
		d.Kubernetes = DestinationKubernetes{
			CA: template.Kubernetes.Ca,
		}
	}

	automaticLabels := []string{
		string(template.Kind),
	}

	for _, l := range append(template.Labels, automaticLabels...) {
		d.Labels = append(d.Labels, Label{Value: l})
	}

	return nil
}
