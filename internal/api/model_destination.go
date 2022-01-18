package api

import (
	"github.com/infrahq/infra/uid"
)

type Destination struct {
	ID       uid.ID `json:"id"`
	UniqueID string `json:"uniqueID" form:"uniqueID"`
	Name     string `json:"name" form:"name"`
	// created time in seconds since 1970-01-01
	Created int64 `json:"created"`
	// updated time in seconds since 1970-01-01
	Updated    int64                 `json:"updated"`
	Connection DestinationConnection `json:"connection"`
}

type DestinationConnection struct {
	URL string `json:"url" validate:"required"`
	CA  string `json:"ca"`
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
	ID         uid.ID                `uri:"id" json:"-" validate:"required"`
	Name       string                `json:"name" validate:"required"`
	UniqueID   string                `json:"uniqueID"`
	Connection DestinationConnection `json:"connection"`
}
