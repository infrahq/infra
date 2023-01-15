package access

import (
	"errors"
	"fmt"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

func ListOrganizations(rCtx RequestContext, name string, pg *data.Pagination) ([]models.Organization, error) {
	err := IsAuthorized(rCtx, models.InfraSupportAdminRole)
	if err == nil {
		return data.ListOrganizations(rCtx.DBTxn, data.ListOrganizationsOptions{
			ByName:     name,
			Pagination: pg,
		})
	}
	err = HandleAuthErr(err, "organizations", "list", models.InfraSupportAdminRole)

	// TODO:
	//    * Consider allowing a user to list their own organization

	return nil, err
}

func GetOrganization(rCtx RequestContext, id uid.ID) (*models.Organization, error) {
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

func CreateOrganization(rCtx RequestContext, org *models.Organization) error {
	err := IsAuthorized(rCtx, models.InfraSupportAdminRole)
	if err != nil {
		return HandleAuthErr(err, "organizations", "create", models.InfraSupportAdminRole)
	}

	return data.CreateOrganization(rCtx.DBTxn, org)
}

func DeleteOrganization(rCtx RequestContext, id uid.ID) error {
	err := IsAuthorized(rCtx, models.InfraSupportAdminRole)
	if err != nil {
		return HandleAuthErr(err, "organizations", "delete", models.InfraSupportAdminRole)
	}

	return data.DeleteOrganization(rCtx.DBTxn, id)
}

// DomainAvailable is needed to check if an org domain is available before completing social sign-up
func DomainAvailable(rCtx RequestContext, domain string) error {
	_, err := data.GetOrganization(rCtx.DBTxn, data.GetOrganizationOptions{ByDomain: domain})
	if err != nil {
		if !errors.Is(err, internal.ErrNotFound) {
			return fmt.Errorf("check domain available: %w", err)
		}
		return nil // not found, so available
	}
	return data.UniqueConstraintError{Table: "organization", Column: "domain", Value: domain}
}

func UpdateOrganization(rCtx RequestContext, org *models.Organization) error {
	if user := rCtx.Authenticated.User; user != nil && user.OrganizationID == org.ID {
		// admins may update their own org
		err := IsAuthorized(rCtx, models.InfraAdminRole)
		if err != nil {
			return HandleAuthErr(err, "organization", "update", models.InfraAdminRole)
		}
		return data.UpdateOrganization(rCtx.DBTxn, org)
	}
	return fmt.Errorf("%w: %s", ErrNotAuthorized, "you may only update your own organization")
}
