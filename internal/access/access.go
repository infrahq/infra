package access

import (
	"errors"
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

func getDB(c *gin.Context) *gorm.DB {
	db, ok := c.MustGet("db").(*gorm.DB)
	if !ok {
		return nil
	}

	return db
}

// hasAuthorization checks if a caller is the owner of a resource before checking if they have an approprite role to access it
func hasAuthorization(c *gin.Context, requestedResource uid.ID, isResourceOwner func(c *gin.Context, requestedResourceID uid.ID) (bool, error), oneOfRoles ...string) (*gorm.DB, error) {
	owner, err := isResourceOwner(c, requestedResource)
	if err != nil {
		return nil, fmt.Errorf("owner lookup: %w", err)
	}

	if owner {
		return getDB(c), nil
	}

	return RequireInfraRole(c, oneOfRoles...)
}

const ResourceInfraAPI = "infra"

// RequireInfraRole checks that the identity in the context can perform an action on a resource based on their granted roles
func RequireInfraRole(c *gin.Context, oneOfRoles ...string) (*gorm.DB, error) {
	db := getDB(c)

	identity := AuthenticatedIdentity(c)
	if identity == nil {
		return nil, fmt.Errorf("no active identity")
	}

	for _, role := range oneOfRoles {
		ok, err := Can(db, identity.PolyID(), role, ResourceInfraAPI)
		if err != nil {
			return nil, err
		}

		if ok {
			return db, nil
		}
	}

	// check if they belong to a group that is authorized
	groups, err := data.ListGroups(db, &models.Pagination{}, data.ByGroupMember(identity.ID))
	if err != nil {
		return nil, fmt.Errorf("auth user groups: %w", err)
	}

	for _, group := range groups {
		for _, role := range oneOfRoles {
			ok, err := Can(db, group.PolyID(), role, ResourceInfraAPI)
			if err != nil {
				return nil, err
			}

			if ok {
				return db, nil
			}
		}
	}

	return nil, ErrNotAuthorized
}

var ErrNotAuthorized = errors.New("not authorized")

// AuthorizationError indicates that the user who performed the operation does
// not have the required role.
type AuthorizationError struct {
	Resource      string
	Operation     string
	RequiredRoles []string
}

func (e AuthorizationError) Error() string {
	var roles strings.Builder
	switch len(e.RequiredRoles) {
	case 1:
		roles.WriteString(e.RequiredRoles[0])
	default:
		for i, role := range e.RequiredRoles {
			roles.WriteString(role)
			switch {
			case i+1 == len(e.RequiredRoles)-1:
				roles.WriteString(", or ")
			case i+1 != len(e.RequiredRoles):
				roles.WriteString(", ")
			}
		}
	}
	return fmt.Sprintf("you do not have permission to %v %v, requires role %v",
		e.Operation, e.Resource, roles.String())
}

func (e AuthorizationError) Is(other error) bool {
	// nolint:errorlint // comparing with == is correct here, the caller uses Unwrap.
	return other == ErrNotAuthorized
}

func HandleAuthErr(err error, resource, operation string, roles ...string) error {
	if !errors.Is(err, ErrNotAuthorized) {
		return err
	}
	return AuthorizationError{
		Resource:      resource,
		Operation:     operation,
		RequiredRoles: roles,
	}
}

// Can checks if an identity has a privilege that means it can perform an action on a resource
func Can(db *gorm.DB, identity uid.PolymorphicID, privilege, resource string) (bool, error) {
	grants, err := data.ListGrants(db, &models.Pagination{}, data.BySubject(identity), data.ByPrivilege(privilege), data.ByResource(resource))
	if err != nil {
		return false, fmt.Errorf("has grants: %w", err)
	}

	return len(grants) > 0, nil
}
