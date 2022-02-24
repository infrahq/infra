package access

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/uid"
)

const (
	AdminRole     = "admin"
	ViewRole      = "view"
	UserRole      = "user"
	ConnectorRole = "connector"
)

func getDB(c *gin.Context) *gorm.DB {
	return c.MustGet("db").(*gorm.DB)
}

// requireInfraRole checks that the identity in the context can perform an action on a resource based on their granted roles
func requireInfraRole(c *gin.Context, oneOfRoles ...string) (*gorm.DB, error) {
	db := getDB(c)

	identity := uid.CurrentIdentity(c)
	if identity == nil {
		return nil, fmt.Errorf("no active identity")
	}

	for _, role := range oneOfRoles {
		ok, err := Can(db, *identity, role, "infra")
		if err != nil {
			return nil, err
		}

		if ok {
			return db, nil
		}
	}

	user := CurrentUser(c)

	if user != nil {
		// check if they belong to a group that is authorized
		groups, err := data.ListUserGroups(db, user.ID)
		if err != nil {
			return nil, fmt.Errorf("auth user groups: %w", err)
		}

		for _, group := range groups {
			for _, role := range oneOfRoles {
				ok, err := Can(db, group.PolymorphicIdentifier(), role, "infra")
				if err != nil {
					return nil, err
				}

				if ok {
					return db, nil
				}
			}
		}
	}

	return nil, fmt.Errorf("%w: requestor does not have required grant", internal.ErrForbidden)
}

// Can checks if an identity has a privilege that means it can perform an action on a resource
func Can(db *gorm.DB, identity uid.PolymorphicID, privilege, resource string) (bool, error) {
	grants, err := data.ListGrants(db, data.ByIdentity(identity), data.ByPrivilege(privilege), data.ByResource(resource))
	if err != nil {
		return false, fmt.Errorf("has grants: %w", err)
	}

	for _, grant := range grants {
		if grant.Matches(identity, privilege, resource) {
			return true, nil
		}
	}

	return false, nil
}
