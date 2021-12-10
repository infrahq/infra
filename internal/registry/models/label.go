package models

import (
	"github.com/google/uuid"
)

type Label struct {
	Model

	Value string

	DestinationID uuid.UUID
	GrantID       uuid.UUID
}
