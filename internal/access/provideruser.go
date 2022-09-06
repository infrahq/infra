package access

import (
	"github.com/gin-gonic/gin"

	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
)

// TODO: remove this, it should be part of creating an identity
func CreateProviderUser(c *gin.Context, provider *models.Provider, ident *models.Identity) (*models.ProviderUser, error) {
	// does not need authorization check, this function should only be called internally
	db := getDB(c)

	return data.CreateProviderUser(db, provider, ident)
}
