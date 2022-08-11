package access

import (
	"errors"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

func GetGrant(c *gin.Context, id uid.ID) (*models.Grant, error) {
	db, err := RequireInfraRole(c, models.InfraAdminRole)
	if err != nil {
		return nil, HandleAuthErr(err, "grant", "get", models.InfraAdminRole)
	}

	return data.GetGrant(db, data.ByID(id))
}

func ListGrants(c *gin.Context, subject uid.PolymorphicID, resource string, privilege string, inherited bool, showSystem bool, p *models.Pagination) ([]models.Grant, error) {
	selectors := []data.SelectorFunc{
		data.ByOptionalResource(resource),
		data.ByOptionalPrivilege(privilege),
	}

	roles := []string{models.InfraAdminRole, models.InfraViewRole, models.InfraConnectorRole}
	db, err := RequireInfraRole(c, roles...)
	err = HandleAuthErr(err, "grants", "list", roles...)
	if errors.Is(err, ErrNotAuthorized) {
		// Allow an authenticated identity to view their own grants
		db = getDB(c)
		subjectID, err2 := subject.ID()
		if err2 != nil {
			// user is only allowed to select their own grants, so if the subject is missing or invalid, this is an access error
			return nil, err
		}
		identity := AuthenticatedIdentity(c)
		switch {
		case identity == nil:
			return nil, err
		case subject.IsIdentity() && identity.ID == subjectID:
			if inherited {
				selectors = append(selectors, data.GrantsInheritedBySubject(subject))
			} else {
				selectors = append(selectors, data.BySubject(subject))
			}
			return data.ListGrants(db, p, selectors...)
		case subject.IsGroup() && userInGroup(db, identity.ID, subjectID):
			if inherited {
				selectors = append(selectors, data.GrantsInheritedBySubject(subject))
			} else {
				selectors = append(selectors, data.BySubject(subject))
			}
			return data.ListGrants(db, p, selectors...)
		default:
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}

	if inherited && len(subject) > 0 {
		selectors = append(selectors, data.GrantsInheritedBySubject(subject))
	} else {
		selectors = append(selectors, data.ByOptionalSubject(subject))
	}

	if !showSystem {
		selectors = append(selectors, data.NotPrivilege(models.InfraConnectorRole))
	}

	return data.ListGrants(db, p, selectors...)
}

func userInGroup(db *gorm.DB, authnUserID uid.ID, groupID uid.ID) bool {
	groups, err := data.ListGroups(db, &models.Pagination{Limit: 1}, data.ByGroupMember(authnUserID), data.ByID(groupID))
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
	var db *gorm.DB
	var err error

	if grant.Privilege == models.InfraSupportAdminRole && grant.Resource == ResourceInfraAPI {
		db, err = RequireInfraRole(c, models.InfraSupportAdminRole)
	} else {
		db, err = RequireInfraRole(c, models.InfraAdminRole)
	}

	if err != nil {
		return HandleAuthErr(err, "grant", "create", grant.Privilege)
	}

	// TODO: CreatedBy should be set automatically
	creator := AuthenticatedIdentity(c)
	grant.CreatedBy = creator.ID

	return data.CreateGrant(db, grant)
}

func DeleteGrant(c *gin.Context, id uid.ID) error {
	db, err := RequireInfraRole(c, models.InfraAdminRole)
	if err != nil {
		return HandleAuthErr(err, "grant", "delete", models.InfraAdminRole)
	}

	return data.DeleteGrants(db, data.ByID(id))
}
