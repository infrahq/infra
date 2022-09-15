package access

import (
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

func ListOrganizations(c *gin.Context, name string, pg *data.Pagination) ([]models.Organization, error) {
	selectors := []data.SelectorFunc{}
	if name != "" {
		selectors = append(selectors, data.ByName(name))
	}

	db, err := RequireInfraRole(c, models.InfraSupportAdminRole)
	if err == nil {
		return data.ListOrganizations(db, pg, selectors...)
	}
	err = HandleAuthErr(err, "organizations", "list", models.InfraSupportAdminRole)

	// TODO:
	//    * Consider allowing a user to list their own organization

	return nil, err
}

func GetOrganization(c *gin.Context, id uid.ID) (*models.Organization, error) {
	db, err := RequireInfraRole(c, models.InfraSupportAdminRole)
	if err != nil {
		return nil, HandleAuthErr(err, "organizations", "get", models.InfraSupportAdminRole)
	}

	return data.GetOrganization(db, data.ByID(id))
}

func CreateOrganization(c *gin.Context, org *models.Organization) error {
	db, err := RequireInfraRole(c, models.InfraSupportAdminRole)
	if err != nil {
		return HandleAuthErr(err, "organizations", "create", models.InfraSupportAdminRole)
	}

	return data.CreateOrganization(db, org)
}

func DeleteOrganization(c *gin.Context, id uid.ID) error {
	db, err := RequireInfraRole(c, models.InfraSupportAdminRole)
	if err != nil {
		return HandleAuthErr(err, "organizations", "delete", models.InfraSupportAdminRole)
	}

	return data.DeleteOrganizations(db, data.ByID(id))
}

func SanitizedDomain(subDomain, serverBaseDomain string) string {
	return strings.ToLower(subDomain) + "." + serverBaseDomain
}
