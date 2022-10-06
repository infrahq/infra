package access

import (
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

// isOrganizationSelf is used by authorization checks to see if the calling identity is requesting their own organization
func isOrganizationSelf(c *gin.Context, orgID uid.ID) (bool, error) {
	org := GetRequestContext(c).Authenticated.Organization
	return org != nil && org.ID == orgID, nil
}

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
	roles := []string{models.InfraSupportAdminRole}

	// If the user is in the org, allow them to call this endpoint, otherwise they must be
	// an InfraSupportAdmin.
	db, err := hasAuthorization(c, id, isOrganizationSelf, roles...)
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
