package api

import (
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
}

type DestinationConnection struct {
	URL string `json:"url" validate:"required" example:"aa60eexample.us-west-2.elb.amazonaws.com"`
	CA  PEM    `json:"ca" example:"-----BEGIN CERTIFICATE-----\nMIIDNTCCAh2gAwIBAgIRALRetnpcTo9O3V2fAK3ix+c\n-----END CERTIFICATE-----\n"`
}

type ListDestinationsRequest struct {
	Name     string `form:"name"`
	UniqueID string `form:"unique_id"`
}

type CreateDestinationRequest struct {
	UniqueID   string                `json:"uniqueID"`
	Name       string                `json:"name" validate:"required"`
	Connection DestinationConnection `json:"connection"`

	Resources []string `json:"resources"`
	Roles     []string `json:"roles"`
}

type UpdateDestinationRequest struct {
	ID         uid.ID                `uri:"id" json:"-" validate:"required"`
	Name       string                `json:"name" validate:"required"`
	UniqueID   string                `json:"uniqueID"`
	Connection DestinationConnection `json:"connection"`

	Resources []string `json:"resources"`
	Roles     []string `json:"roles"`
}
