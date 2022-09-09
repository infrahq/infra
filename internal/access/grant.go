package access

import (
	"errors"

	"github.com/gin-gonic/gin"

	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

func GetGrant(c *gin.Context, id uid.ID) (*models.Grant, error) {
	db, err := RequireInfraRole(c, models.InfraAdminRole)
	if err != nil {
		return nil, HandleAuthErr(err, "grant", "get", models.InfraAdminRole)
	}

	return data.GetGrant(db, data.GetGrantOptions{ByID: id})
}

func ListGrants(c *gin.Context, subject uid.PolymorphicID, resource string, privilege string, inherited bool, showSystem bool, p *data.Pagination) ([]models.Grant, error) {
	rCtx := GetRequestContext(c)

	roles := []string{models.InfraAdminRole, models.InfraViewRole, models.InfraConnectorRole}
	_, err := RequireInfraRole(c, roles...)
	err = HandleAuthErr(err, "grants", "list", roles...)
	if errors.Is(err, ErrNotAuthorized) {
		// Allow an authenticated identity to view their own grants
		subjectID, _ := subject.ID() // zero value will never match a user
		switch {
		case rCtx.Authenticated.User == nil:
			return nil, err
		case subject.IsIdentity() && rCtx.Authenticated.User.ID == subjectID:
			// authorized because the request is for their own grants
		case subject.IsGroup() && userInGroup(rCtx.DBTxn, rCtx.Authenticated.User.ID, subjectID):
			// authorized because the request is for grants of a group they belong to
		default:
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}

	opts := data.ListGrantsOptions{
		ByResource:                 resource,
		ByPrivilege:                privilege,
		BySubject:                  subject,
		ExcludeConnectorGrant:      !showSystem,
		IncludeInheritedFromGroups: inherited,
		Pagination:                 p,
	}
	return data.ListGrants(rCtx.DBTxn, opts)
}

func userInGroup(db data.GormTxn, authnUserID uid.ID, groupID uid.ID) bool {
	groups, err := data.ListGroups(db, &data.Pagination{Limit: 1}, data.ByGroupMember(authnUserID), data.ByID(groupID))
	if err != nil {
		return false
	}

	for _, g := range groups {
		if g.ID == groupID {
			return true
		}
	}
	return false
}

func CreateGrant(c *gin.Context, grant *models.Grant) error {
	rCtx := GetRequestContext(c)

	var err error
	if grant.Privilege == models.InfraSupportAdminRole && grant.Resource == ResourceInfraAPI {
		_, err = RequireInfraRole(c, models.InfraSupportAdminRole)
	} else {
		_, err = RequireInfraRole(c, models.InfraAdminRole)
	}

	if err != nil {
		return HandleAuthErr(err, "grant", "create", grant.Privilege)
	}

	// TODO: CreatedBy should be set automatically
	grant.CreatedBy = rCtx.Authenticated.User.ID

	return data.CreateGrant(rCtx.DBTxn, grant)
}

func DeleteGrant(c *gin.Context, id uid.ID) error {
	db, err := RequireInfraRole(c, models.InfraAdminRole)
	if err != nil {
		return HandleAuthErr(err, "grant", "delete", models.InfraAdminRole)
	}

	return data.DeleteGrants(db, data.DeleteGrantsOptions{ByID: id})
}
