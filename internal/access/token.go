package access

import (
	"fmt"

	"github.com/gin-gonic/gin"

	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
)

func CreateToken(c *gin.Context) (token *models.Token, err error) {
	identity := CurrentIdentity(c)
	if identity == nil {
		return nil, fmt.Errorf("no active identity")
	}

	// does not need authorization check, limited to calling identity
	db := getDB(c)

	return data.CreateIdentityToken(db, identity.ID)
}
