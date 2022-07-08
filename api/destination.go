package api

import (
	"github.com/infrahq/infra/internal/validate"
	"github.com/infrahq/infra/uid"
)

type Destination struct {
	ID         uid.ID                `json:"id"`
	UniqueID   string                `json:"uniqueID" form:"uniqueID" example:"94c2c570a20311180ec325fd56"`
	Name       string                `json:"name" form:"name"`
	Created    Time                  `json:"created"`
	Updated    Time                  `json:"updated"`
	Connection DestinationConnection `json:"connection"`

	Resources []string `json:"resources"`
	Roles     []string `json:"roles"`

	LastSeen  Time `json:"lastSeen"`
	Connected bool `json:"connected"`

	Version string `json:"version"`
}

type DestinationConnection struct {
	URL string `json:"url" example:"aa60eexample.us-west-2.elb.amazonaws.com"`
	CA  PEM    `json:"ca" example:"-----BEGIN CERTIFICATE-----\nMIIDNTCCAh2gAwIBAgIRALRetnpcTo9O3V2fAK3ix+c\n-----END CERTIFICATE-----\n"`
}

func (r DestinationConnection) ValidationRules() []validate.ValidationRule {
	return []validate.ValidationRule{
		validate.Required("url", r.URL),
	}
}

type ListDestinationsRequest struct {
	Name     string `form:"name"`
	UniqueID string `form:"unique_id"`
	PaginationRequest
}

func (r ListDestinationsRequest) ValidationRules() []validate.ValidationRule {
	return nil
}

type CreateDestinationRequest struct {
	UniqueID   string                `json:"uniqueID"`
	Name       string                `json:"name"`
	Version    string                `json:"version"`
	Connection DestinationConnection `json:"connection"`

	Resources []string `json:"resources"`
	Roles     []string `json:"roles"`
}

func (r CreateDestinationRequest) ValidationRules() []validate.ValidationRule {
	return []validate.ValidationRule{
		ValidateName(r.Name),
		validate.Required("name", r.Name),
		validate.Required("uniqueID", r.UniqueID),
	}
}

type UpdateDestinationRequest struct {
	ID         uid.ID                `uri:"id" json:"-"`
	Name       string                `json:"name"`
	UniqueID   string                `json:"uniqueID"`
	Version    string                `json:"version"`
	Connection DestinationConnection `json:"connection"`

	Resources []string `json:"resources"`
	Roles     []string `json:"roles"`
}

func (r UpdateDestinationRequest) ValidationRules() []validate.ValidationRule {
	return []validate.ValidationRule{
		ValidateName(r.Name),
		validate.Required("name", r.Name),
		validate.Required("id", r.ID),
		validate.Required("uniqueID", r.UniqueID),
	}
}
