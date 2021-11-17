package data

import (
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/infrahq/infra/internal"
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

func CreateDestination(db *gorm.DB, destination *Destination) (*Destination, error) {
	if err := add(db, &Destination{}, destination, &Destination{}); err != nil {
		return nil, err
	}

	return destination, nil
}

func CreateOrUpdateDestination(db *gorm.DB, destination *Destination, condition interface{}) (*Destination, error) {
	existing, err := GetDestination(db, condition)
	if err != nil {
		if !errors.Is(err, internal.ErrNotFound) {
			return nil, err
		}

		if _, err := CreateDestination(db, destination); err != nil {
			return nil, err
		}

		return destination, nil
	}

	if err := update(db, &Destination{}, destination, db.Where(existing, "id")); err != nil {
		return nil, err
	}

	switch destination.Kind {
	case DestinationKindKubernetes:
		if err := db.Model(existing).Association("Kubernetes").Replace(&destination.Kubernetes); err != nil {
			return nil, err
		}
	}

	if err := db.Model(existing).Association("Labels").Replace(&destination.Labels); err != nil {
		return nil, err
	}

	return GetDestination(db, db.Where(existing, "id"))
}

func GetDestination(db *gorm.DB, condition interface{}) (*Destination, error) {
	var destination Destination
	if err := get(db, &Destination{}, &destination, condition); err != nil {
		return nil, err
	}

	return &destination, nil
}

func ListDestinations(db *gorm.DB, condition interface{}) ([]Destination, error) {
	destinations := make([]Destination, 0)
	if err := list(db, &Destination{}, &destinations, condition); err != nil {
		return nil, err
	}

	return destinations, nil
}

func DeleteDestinations(db *gorm.DB, condition interface{}) error {
	toDelete, err := ListDestinations(db, condition)
	if err != nil {
		return err
	}

	if len(toDelete) > 0 {
		ids := make([]uuid.UUID, 0)
		for _, g := range toDelete {
			ids = append(ids, g.ID)
		}

		return remove(db, &Destination{}, ids)
	}

	return nil
}
