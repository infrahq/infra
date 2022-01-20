package models

import (
	"github.com/infrahq/infra/uuid"
)

type Label struct {
	Model

	Value string

	DestinationID uuid.UUID
}
