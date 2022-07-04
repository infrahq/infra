package access

import (
	"github.com/gin-gonic/gin"

	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

func ListOrganizations(c *gin.Context, name string, pg models.Pagination) ([]models.Organization, error) {
	var selectors = []data.SelectorFunc{data.ByPagination(pg)}
	if name != "" {
		selectors = append(selectors, data.ByName(name))
	}

	roles := []string{models.InfraAdminRole}
	db, err := RequireInfraRole(c, roles...)
	if err == nil {
		return data.ListOrganizations(db, selectors...)
	}
	err = HandleAuthErr(err, "organizations", "list", roles...)

	// TODO:
	//    * Consider allowing a user to list their own organization

	return nil, err
}

func GetOrganization(c *gin.Context, id uid.ID) (*models.Organization, error) {
	roles := []string{models.InfraAdminRole}
	db, err := RequireInfraRole(c, roles...)
	if err != nil {
		return nil, HandleAuthErr(err, "organizations", "get", roles...)
	}

	return data.GetOrganization(db, data.ByID(id))
}

func CreateOrganization(c *gin.Context, org *models.Organization) error {
	roles := []string{models.InfraAdminRole}
	db, err := RequireInfraRole(c, roles...)
	if err != nil {
		return HandleAuthErr(err, "organizations", "create", roles...)
	}

	return data.CreateOrganization(db, org)
}

func DeleteOrganization(c *gin.Context, id uid.ID) error {
	roles := []string{models.InfraAdminRole}
	db, err := RequireInfraRole(c, roles...)
	if err != nil {
		return HandleAuthErr(err, "organizations", "delete", roles...)
	}

	selectors := []data.SelectorFunc{
		data.ByID(id),
	}

	return data.DeleteOrganizations(db, selectors...)
}
