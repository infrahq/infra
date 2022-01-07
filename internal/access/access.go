package access

import (
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/registry/models"
)

type Permission string

const (
	PermissionAll       Permission = "*"
	PermissionAllInfra  Permission = "infra.*"
	PermissionAllCreate Permission = "infra.*.create"
	PermissionAllRead   Permission = "infra.*.read"
	PermissionAllUpdate Permission = "infra.*.update"
	PermissionAllDelete Permission = "infra.*.delete"
)

// requireAuthorizationWithCheck checks first if the customCheckFunc returns true.
// If so, the user is an owner of the object or has some direct right to do the action requested,
// and thus the rest of the permission checks can be skipped.
// This should be used conservatively for things like record ownership.
// If customCheckFunc returns false, the requestor must prove access via permissions.
// Note that customCheckFunc is only called when there is a currentUser present in the request.
func requireAuthorizationWithCheck(c *gin.Context, require Permission, customCheckFunc func(currUser *models.User) bool) (*gorm.DB, error) {
	db := getDB(c)

	user := currentUser(c)

	if user != nil && customCheckFunc(user) {
		return db, nil
	}

	return requireAuthorization(c, require)
}

func getDB(c *gin.Context) *gorm.DB {
	return c.MustGet("db").(*gorm.DB)
}

func hasAuthorization(c *gin.Context, requires ...Permission) (bool, error) {
	if len(requires) == 0 {
		return true, nil
	}

	permissionStr, ok := c.MustGet("permissions").(string)
	if !ok {
		return false, fmt.Errorf("%w: requestor has no permissions", internal.ErrForbidden)
	}
	permissions := strings.Split(permissionStr, " ")

outer:
	for _, required := range requires {
		for _, p := range permissions {
			if hasRequiredPermission(p, string(required)) {
				continue outer
			}
		}
		return false, fmt.Errorf("missing permission %q: %w", required, internal.ErrForbidden)
	}

	return true, nil
}

// requireAuthorization checks that the context has the permissions required to perform the action
func requireAuthorization(c *gin.Context, requires ...Permission) (*gorm.DB, error) {
	db := getDB(c)

	ok, err := hasAuthorization(c, requires...)
	if !ok {
		return nil, err
	}

	return db, nil
}

// hasRequiredPermission checks if a given permission grants a required permission
func hasRequiredPermission(permission, required string) bool {
	if Permission(permission) == PermissionAll || Permission(permission) == PermissionAllInfra {
		return true
	} else if permission == required {
		return true
	}

	parts := strings.Split(permission, ".")
	for i, part := range strings.Split(required, ".") {
		if part == parts[i] || parts[i] == "*" {
			continue
		}

		return false
	}

	return true
}

// AllRequired checks if a a set of permissions contains all the required permissions
func AllRequired(permissions, required []string) bool {
	for _, req := range required {
		granted := false
		for _, p := range permissions {
			granted = hasRequiredPermission(p, req)
			if granted {
				break
			}
		}

		if !granted {
			return false
		}
	}

	return true
}
