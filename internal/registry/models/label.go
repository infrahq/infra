package models

import "github.com/infrahq/infra/uid"

type Label struct {
	Model

	Value string

	DestinationID uid.ID
}
