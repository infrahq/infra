package access

import (
	"fmt"

	"github.com/gin-gonic/gin"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

func CreateProvider(c *gin.Context, provider *models.Provider) error {
	db, err := RequireInfraRole(c, models.InfraAdminRole)
	if err != nil {
		return HandleAuthErr(err, "provider", "create", models.InfraAdminRole)
	}

	return data.CreateProvider(db, provider)
}

func GetProvider(c *gin.Context, id uid.ID) (*models.Provider, error) {
	db := getDB(c)

	return data.GetProvider(db, data.ByID(id))
}

func ListProviders(c *gin.Context, name string, excludeByName []string, pg models.Pagination) ([]models.Provider, error) {
	db := getDB(c)

	selectors := []data.SelectorFunc{
		data.ByOptionalName(name),
		data.ByPagination(pg),
	}

	for _, exclude := range excludeByName {
		selectors = append(selectors, data.NotName(exclude))
	}

	return data.ListProviders(db, selectors...)
}

func SaveProvider(c *gin.Context, provider *models.Provider) error {
	db, err := RequireInfraRole(c, models.InfraAdminRole)
	if err != nil {
		return HandleAuthErr(err, "provider", "update", models.InfraAdminRole)
	}
	if InfraProvider(c).ID == provider.ID {
		return fmt.Errorf("%w: the infra provider can not be modified", internal.ErrBadRequest)
	}

	return data.SaveProvider(db, provider)
}

func DeleteProvider(c *gin.Context, id uid.ID) error {
	db, err := RequireInfraRole(c, models.InfraAdminRole)
	if err != nil {
		return HandleAuthErr(err, "provider", "delete", models.InfraAdminRole)
	}
	if InfraProvider(c).ID == id {
		return fmt.Errorf("%w: the infra provider can not be deleted", internal.ErrBadRequest)
	}

	return data.DeleteProviders(db, data.ByID(id))
}

func InfraProvider(c *gin.Context) *models.Provider {
	db := getDB(c)

	return data.InfraProvider(db)
}
