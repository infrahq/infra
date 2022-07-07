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
	URL string `json:"url" validate:"required" example:"aa60eexample.us-west-2.elb.amazonaws.com"`
	CA  PEM    `json:"ca" example:"-----BEGIN CERTIFICATE-----\nMIIDNTCCAh2gAwIBAgIRALRetnpcTo9O3V2fAK3ix+c\n-----END CERTIFICATE-----\n"`
}

type ListDestinationsRequest struct {
	Name     string `form:"name"`
	UniqueID string `form:"unique_id"`
	PaginationRequest
}

func (r ListDestinationsRequest) ValidationRules() []validate.ValidationRule {
	return r.PaginationRequest.ValidationRules()
}

type CreateDestinationRequest struct {
	UniqueID string `json:"uniqueID" validate:"required"`
	Name     string `json:"name" validate:"required"`
	Version  string `json:"version"`

	Connection DestinationConnection `json:"connection"`

	Resources []string `json:"resources"`
	Roles     []string `json:"roles"`
}

type UpdateDestinationRequest struct {
	ID       uid.ID `uri:"id" json:"-" validate:"required"`
	UniqueID string `json:"uniqueID" validate:"required"`
	Name     string `json:"name" validate:"required"`
	Version  string `json:"version"`

	Connection DestinationConnection `json:"connection"`

	Resources []string `json:"resources"`
	Roles     []string `json:"roles"`
}
