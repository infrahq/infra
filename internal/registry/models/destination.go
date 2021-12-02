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
	if createRequest, ok := from.(*api.DestinationCreateRequest); ok {
		d.Name = createRequest.Name
		d.NodeID = createRequest.NodeID
		d.Kind = DestinationKind(createRequest.Kind)
		d.Endpoint = createRequest.Kubernetes.Endpoint

		switch d.Kind {
		case DestinationKindKubernetes:
			d.Kubernetes = DestinationKubernetes{
				CA: createRequest.Kubernetes.CA,
			}

		}

		automaticLabels := []string{
			string(createRequest.Kind),
		}
	
		for _, l := range append(createRequest.Labels, automaticLabels...) {
			d.Labels = append(d.Labels, Label{Value: l})
		}

		return nil
	}

	return fmt.Errorf("unknown API model")
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
