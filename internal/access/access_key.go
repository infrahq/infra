package access

import (
	"fmt"

	"github.com/gin-gonic/gin"

	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

func ListAccessKeys(c *gin.Context, identityID uid.ID, name string, showExpired bool, p *models.Pagination) ([]models.AccessKey, error) {
	roles := []string{models.InfraAdminRole, models.InfraViewRole}
	db, err := RequireInfraRole(c, roles...)
	if err != nil {
		return nil, HandleAuthErr(err, "access keys", "list", roles...)
	}

	s := []data.SelectorFunc{data.ByOptionalIssuedFor(identityID), data.ByOptionalName(name)}
	if !showExpired {
		s = append(s, data.ByNotExpiredOrExtended())
	}

	return data.ListAccessKeys(db.Preload("IssuedForIdentity"), p, s...)
}

func CreateAccessKey(c *gin.Context, accessKey *models.AccessKey) (body string, err error) {
	db, err := RequireInfraRole(c, models.InfraAdminRole)
	if err != nil {
		return "", HandleAuthErr(err, "access key", "create", models.InfraAdminRole)
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

	return data.DeleteAccessKey(db, id)
}

func DeleteRequestAccessKey(c RequestContext) error {
	// does not need authorization check, this action is limited to the calling key
	return data.DeleteAccessKey(c.DBTxn, c.Authenticated.AccessKey.ID)
}
