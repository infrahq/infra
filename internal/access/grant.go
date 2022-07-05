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

func ListGrants(c *gin.Context, subject uid.PolymorphicID, resource string, privilege string, pg models.Pagination) ([]models.Grant, error) {
	selectors := []data.SelectorFunc{
		data.ByOptionalResource(resource),
		data.ByOptionalPrivilege(privilege),
		data.ByPagination(pg),
	}

	roles := []string{models.InfraAdminRole, models.InfraViewRole, models.InfraConnectorRole}
	db, err := RequireInfraRole(c, roles...)
	if err == nil {
		selectors = append(selectors, data.ByOptionalSubject(subject))
		return data.ListGrants(db, selectors...)
	}
	err = HandleAuthErr(err, "grants", "list", roles...)

	if errors.Is(err, ErrNotAuthorized) {
		// Allow an authenticated identity to view their own grants
		db := getDB(c)
		subjectID, _ := subject.ID()
		identity := AuthenticatedIdentity(c)
		switch {
		case identity == nil:
			return nil, err
		case subject.IsIdentity() && identity.ID == subjectID:
			selectors = append(selectors, data.BySubject(subject))
			return data.ListGrants(db, selectors...)
		case subject.IsGroup() && userInGroup(db, identity.ID, subjectID):
			selectors = append(selectors, data.BySubject(subject))
			return data.ListGrants(db, selectors...)
		}
	}

	return nil, err
}

func userInGroup(db *gorm.DB, authnUserID uid.ID, groupID uid.ID) bool {
	groups, err := data.ListGroups(db, data.ByGroupMember(authnUserID), data.ByID(groupID))
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
	db, err := RequireInfraRole(c, models.InfraAdminRole)
	if err != nil {
		return HandleAuthErr(err, "grant", "create", models.InfraAdminRole)
	}

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
