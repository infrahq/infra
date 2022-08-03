package access

import (
	"fmt"

	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
)

func CreateToken(c RequestContext) (token *models.Token, err error) {
	// does not need authorization check, limited to calling identity
	if c.Authenticated.User == nil {
		return nil, fmt.Errorf("no active identity")
	}

	return data.CreateIdentityToken(c.DBTxn, c.Authenticated.User.ID)
}
