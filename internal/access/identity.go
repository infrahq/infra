package access

import (
	"github.com/gin-gonic/gin"

	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

// isIdentitySelf is used by authorization checks to see if the calling identity is requesting their own attributes
func isIdentitySelf(rCtx RequestContext, opts data.GetIdentityOptions) bool {
	identity := rCtx.Authenticated.User

	if identity == nil {
		return false
	}

	switch {
	case opts.ByID != 0:
		return identity.ID == opts.ByID
	case opts.ByName != "":
		return identity.Name == opts.ByName
	}

	return false
}

func GetIdentity(c *gin.Context, opts data.GetIdentityOptions) (*models.Identity, error) {
	rCtx := GetRequestContext(c)
	// anyone can get their own user data
	if !isIdentitySelf(rCtx, opts) {
		roles := []string{models.InfraAdminRole, models.InfraViewRole, models.InfraConnectorRole}
		err := IsAuthorized(rCtx, roles...)
		if err != nil {
			return nil, HandleAuthErr(err, "user", "get", roles...)
		}
	}

	return data.GetIdentity(rCtx.DBTxn, opts)
}

func CreateIdentity(c *gin.Context, identity *models.Identity) error {
	db, err := RequireInfraRole(c, models.InfraAdminRole)
	if err != nil {
		return HandleAuthErr(err, "user", "create", models.InfraAdminRole)
	}

	return data.CreateIdentity(db, identity)
}

func DeleteIdentity(c *gin.Context, id uid.ID) error {
	db, err := RequireInfraRole(c, models.InfraAdminRole)
	if err != nil {
		return HandleAuthErr(err, "user", "delete", models.InfraAdminRole)
	}

	opts := data.DeleteIdentitiesOptions{
		ByProviderID: data.InfraProvider(db).ID,
		ByID:         id,
	}

	return data.DeleteIdentities(db, opts)
}

func ListIdentities(c *gin.Context, opts data.ListIdentityOptions) ([]models.Identity, error) {
	roles := []string{models.InfraAdminRole, models.InfraViewRole, models.InfraConnectorRole}
	db, err := RequireInfraRole(c, roles...)
	if err != nil {
		return nil, HandleAuthErr(err, "users", "list", roles...)
	}
	return data.ListIdentities(db, opts)
}
