package access

import (
	"fmt"

	"github.com/gin-gonic/gin"

	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

func ListAccessKeys(c *gin.Context, identityID uid.ID, name string, showExpired bool, p *data.Pagination) ([]models.AccessKey, error) {
	var db data.GormTxn
	var err error

	if identityID == GetRequestContext(c).Authenticated.User.ID {
		db = getDB(c)
	} else {
		roles := []string{models.InfraAdminRole, models.InfraViewRole}
		db, err = RequireInfraRole(c, roles...)
		if err != nil {
			return nil, HandleAuthErr(err, "access keys", "list", roles...)
		}
	}

	opts := data.ListAccessKeyOptions{
		Pagination:     p,
		IncludeExpired: showExpired,
		ByIssuedForID:  identityID,
		ByName:         name,
	}
	return data.ListAccessKeys(db, opts)
}

func CreateAccessKey(c *gin.Context, accessKey *models.AccessKey) (body string, err error) {
	var db data.GormTxn

	if accessKey.IssuedFor == GetRequestContext(c).Authenticated.User.ID {
		db = getDB(c) // can create access keys for yourself.
	} else {
		db, err = RequireInfraRole(c, models.InfraAdminRole)
		if err != nil {
			return "", HandleAuthErr(err, "access key", "create", models.InfraAdminRole)
		}
	}

	if accessKey.ProviderID == 0 {
		accessKey.ProviderID = data.InfraProvider(db).ID
	}

	body, err = data.CreateAccessKey(db, accessKey)
	if err != nil {
		return "", fmt.Errorf("create token: %w", err)
	}

	return body, err
}

func DeleteAccessKey(c *gin.Context, id uid.ID) error {
	db, err := RequireInfraRole(c, models.InfraAdminRole)
	if err != nil {
		return HandleAuthErr(err, "access key", "delete", models.InfraAdminRole)
	}

	return data.DeleteAccessKeys(db, data.DeleteAccessKeysOptions{ByID: id})
}

func DeleteRequestAccessKey(c RequestContext) error {
	// does not need authorization check, this action is limited to the calling key

	id := c.Authenticated.AccessKey.ID
	return data.DeleteAccessKeys(c.DBTxn, data.DeleteAccessKeysOptions{ByID: id})
}
