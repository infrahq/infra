package claims

import (
	"github.com/infrahq/infra/uid"
)

type Custom struct {
	Name           string   `json:"name"`
	Groups         []string `json:"groups"`
	Nonce          string   `json:"nonce"`
	OrganizationID uid.ID   `json:"org"`
}
