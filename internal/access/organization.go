package access

import (
	"errors"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/logging"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

func ListOrganizations(c *gin.Context, name string, pg *data.Pagination) ([]models.Organization, error) {
	db, err := RequireInfraRole(c, models.InfraSupportAdminRole)
	if err == nil {
		return data.ListOrganizations(db, data.ListOrganizationsOptions{
			ByName:     name,
			Pagination: pg,
		})
	}
	err = HandleAuthErr(err, "organizations", "list", models.InfraSupportAdminRole)

	// TODO:
	//    * Consider allowing a user to list their own organization

	return nil, err
}

func GetOrganization(c *gin.Context, id uid.ID) (*models.Organization, error) {
	rCtx := GetRequestContext(c)
	if user := rCtx.Authenticated.User; user != nil && user.OrganizationID == id {
		// request is authorized because the user is a member of the org
	} else {
		roles := []string{models.InfraSupportAdminRole}
		err := IsAuthorized(rCtx, roles...)
		if err != nil {
			return nil, HandleAuthErr(err, "organizations", "get", roles...)
		}
	}

	return data.GetOrganization(rCtx.DBTxn, data.GetOrganizationOptions{ByID: id})
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

	return data.DeleteOrganization(db, id)
}

func SanitizedDomain(subDomain, serverBaseDomain string) string {
	return strings.ToLower(subDomain) + "." + serverBaseDomain
}

// DomainAvailable is needed to check if an org domain is available before completing social sign-up
func DomainAvailable(c *gin.Context, domain string) bool {
	rCtx := GetRequestContext(c)
	_, err := data.GetOrganization(rCtx.DBTxn, data.GetOrganizationOptions{ByDomain: domain})
	if err != nil {
		if !errors.Is(err, internal.ErrNotFound) {
			logging.L.Error().Err(err).Msg("failed to check if organization exists by domain")
		}
		return true // not found, so available
	}
	return false // found, not available
}
