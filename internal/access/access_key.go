package access

import (
	"fmt"

	"github.com/gin-gonic/gin"

	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

func currentAccessKey(c *gin.Context) *models.AccessKey {
	accessKey, ok := c.MustGet("key").(*models.AccessKey)
	if !ok {
		return nil
	}

	return accessKey
}

func ListAccessKeys(c *gin.Context, identityID uid.ID, name string, showExpired bool, pg models.Pagination) ([]models.AccessKey, error) {
	db, err := RequireInfraRole(c, models.InfraAdminRole, models.InfraViewRole)
	if err != nil {
		return nil, err
	}

	s := []data.SelectorFunc{data.ByOptionalIssuedFor(identityID), data.ByOptionalName(name), data.ByPagination(pg)}
	if !showExpired {
		s = append(s, data.ByNotExpiredOrExtended())
	}

	return data.ListAccessKeys(db.Preload("IssuedForIdentity"), s...)
}

func CreateAccessKey(c *gin.Context, accessKey *models.AccessKey) (body string, err error) {
	db, err := RequireInfraRole(c, models.InfraAdminRole)
	if err != nil {
		return "", err
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
		return err
	}

	return data.DeleteAccessKeys(db, data.ByID(id))
}

func DeleteRequestAccessKey(c *gin.Context) error {
	// does not need authorization check, this action is limited to the calling key
	key := currentAccessKey(c)

	db := getDB(c)

	return data.DeleteAccessKey(db, key.ID)
}
