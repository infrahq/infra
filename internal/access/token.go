package access

import (
	"fmt"

	"github.com/gin-gonic/gin"

	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
)

func CreateToken(c *gin.Context) (token *models.Token, err error) {
	identity := AuthenticatedIdentity(c)
	if identity == nil {
		return nil, fmt.Errorf("no active identity")
	}

	// does not need authorization check, limited to calling identity
	db := getDB(c)

	orgID, err := GetCurrentOrgID(c)
	if err != nil {
		return nil, err
	}

	return data.CreateIdentityToken(db, orgID, identity.ID)
}
