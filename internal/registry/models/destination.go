package models

import (
	"fmt"
	"reflect"

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

	return &result
}

func (d *Destination) FromAPI(from interface{}) error {
	if reflect.ValueOf(from).IsNil() {
		return nil
	}

	if request, ok := from.(*api.CreateDestinationRequest); ok {
		d.Name = request.Name
		d.NodeID = request.NodeID
		d.Kind = DestinationKind(request.Kind)

		if kubernetes := request.Kubernetes; kubernetes != nil {
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

	return fmt.Errorf("unknown destination model: " + reflect.TypeOf(from).String())
}
