package access

import (
	"fmt"

	"github.com/gin-gonic/gin"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/logging"
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

	orgSelector, err := GetCurrentOrgSelector(c)
	if err != nil {
		return nil, fmt.Errorf("Couldn't get org for user")
	}

	return data.GetProvider(db, orgSelector, data.ByID(id))
}

func ListProviders(c *gin.Context, name string, orgID uid.ID, excludeByKind []models.ProviderKind, pg models.Pagination) ([]models.Provider, error) {
	db := getDB(c)

	var orgSelector data.SelectorFunc
	var err error

	if orgID == 0 {
		orgSelector, err = GetCurrentOrgSelector(c)
		if err != nil {
			return nil, fmt.Errorf("Couldn't get org for user")
		}
	} else {
		orgSelector = data.ByOrg(orgID)
	}

	selectors := []data.SelectorFunc{
		orgSelector,
		data.ByOptionalName(name),
		data.ByPagination(pg),
	}

	for _, exclude := range excludeByKind {
		selectors = append(selectors, data.NotProviderKind(exclude))
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

	orgID, err := GetCurrentOrgID(c)
	if err != nil {
		return err
	}
	provider.OrganizationID = orgID

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

	orgSelector, err := GetCurrentOrgSelector(c)
	if err != nil {
		return fmt.Errorf("Couldn't get org for user")
	}

	return data.DeleteProviders(db, orgSelector, data.ByID(id))
}

func InfraProvider(c *gin.Context) *models.Provider {
	logging.Infof("!!! INFRA PROVIDER")
	db := getDB(c)

	logging.Infof("got db")
	orgID, err := GetCurrentOrgID(c)
	if err != nil {
		logging.Infof("Couldn't get org ID")
		return nil
	}

	logging.Infof("org ID = %s", orgID)

	return data.InfraProvider(db, orgID)
}
