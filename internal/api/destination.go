package api

import (
	"github.com/infrahq/infra/uid"
)

type Destination struct {
	ID         uid.ID                `json:"id" swaggertype:"string" example:"42FmGLCWzf"`
	UniqueID   string                `json:"uniqueID" form:"uniqueID" example:"94c2c570a20311180ec325fd56"`
	Name       string                `json:"name" form:"name" example:"kubernetes.production"`
	Created    int64                 `json:"created" example:"1646427487"` // created time in seconds since 1970-01-01 00:00:00 UTC
	Updated    int64                 `json:"updated" example:"1646427981"` // updated time in seconds since 1970-01-01 00:00:00 UTC
	Connection DestinationConnection `json:"connection"`
}

type DestinationConnection struct {
	URL string `json:"url" validate:"required" example:"ad60eab86122a.us-west-2.elb.amazonaws.com"`
	CA  string `json:"ca" example:"-----BEGIN CERTIFICATE-----\nMIIDNTCCAh2gAwIBAgIRALRetnpcTo9O3V2fAK3ix+c\n-----END CERTIFICATE-----\n"`
}

type ListDestinationsRequest struct {
	Name     string `form:"name"`
	UniqueID string `form:"unique_id"`
}

type CreateDestinationRequest struct {
	UniqueID   string                `json:"uniqueID"`
	Name       string                `json:"name" validate:"required"`
	Connection DestinationConnection `json:"connection"`
}

type UpdateDestinationRequest struct {
	ID         uid.ID                `uri:"id" json:"-" validate:"required" swaggertype:"string"`
	Name       string                `json:"name" validate:"required"`
	UniqueID   string                `json:"uniqueID"`
	Connection DestinationConnection `json:"connection"`
}
