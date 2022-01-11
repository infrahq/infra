package access

import (
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/logging"
	"github.com/infrahq/infra/internal/registry/data"
	"github.com/infrahq/infra/internal/registry/models"
)

type Permission string

const (
	PermissionAll          Permission = "*"
	PermissionAllAlternate Permission = "infra.*"
	PermissionAllCreate    Permission = "infra.*.create"
	PermissionAllRead      Permission = "infra.*.read"
	PermissionAllUpdate    Permission = "infra.*.update"
	PermissionAllDelete    Permission = "infra.*.delete"
)

// RequireAuthentication checks the bearer token is present and valid then adds its permissions to the context
func RequireAuthentication(c *gin.Context) error {
	db, ok := c.MustGet("db").(*gorm.DB)
	if !ok {
		return fmt.Errorf("no database found in context for authentication")
	}

	header := c.Request.Header.Get("Authorization")

	parts := strings.Split(header, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		return fmt.Errorf("valid token not found in authorization header, expecting the format `Bearer $token`")
	}

	bearer := parts[1]

	if len(bearer) != models.TokenLength {
		return fmt.Errorf("rejected token of invalid length")
	}

	token, err := data.GetToken(db, &models.Token{Key: bearer[:models.TokenKeyLength]})
	if err != nil {
		return fmt.Errorf("could not get token from database, it may not exist: %w", err)
	}

	if err := data.CheckTokenSecret(token, bearer); err != nil {
		return fmt.Errorf("rejected invalid token: %w", err)
	}

	if err := data.CheckTokenExpired(token); err != nil {
		return fmt.Errorf("rejected token: %w", err)
	}

	c.Set("authentication", bearer)

	// token is valid, check where to set permissions from
	if token.UserID != uuid.Nil {
		u, err := data.GetUser(db, db.Where("id = (?)", token.UserID))
		if err != nil {
			return fmt.Errorf("token user lookup: %w", err)
		}

		u.LastSeen = time.Now()
		if _, err := data.UpdateUser(db, u, data.ByUUID(u.ID)); err != nil {
			return fmt.Errorf("user update fail: %w", err)
		}

		logging.S.Debug("user permissions: %s \n", u.Permissions)
		// this token has a parent user, set by their current permissions
		c.Set("permissions", u.Permissions)
	} else if token.APITokenID != uuid.Nil {
		at, err := data.GetAPIToken(db, db.Where("id = (?)", token.APITokenID))
		if err != nil {
			return fmt.Errorf("token api permission lookup: %w", err)
		}
		// this is an API token
		c.Set("permissions", at.Permissions)
	}

	return nil
}

// RequireAuthorization checks that the context has the permissions required to perform the action
func RequireAuthorization(c *gin.Context, require Permission) (*gorm.DB, error) {
	val, ok := c.Get("db")
	if !ok {
		return nil, fmt.Errorf("database not found")
	}

	db, ok := val.(*gorm.DB)
	if !ok {
		return nil, fmt.Errorf("database not found")
	}

	if len(require) == 0 {
		return db, nil
	}

	permissions, ok := c.MustGet("permissions").(string)
	if !ok {
		return nil, internal.ErrForbidden
	}

	for _, p := range strings.Split(permissions, " ") {
		if GrantsPermission(p, string(require)) {
			return db, nil
		}
	}

	return nil, internal.ErrForbidden
}

// GrantsPermission checks if a given permission grants a required permission
func GrantsPermission(permission, require string) bool {
	if Permission(permission) == PermissionAll || Permission(permission) == PermissionAllAlternate {
		return true
	} else if permission == require {
		return true
	}

	parts := strings.Split(permission, ".")
	for i, part := range strings.Split(require, ".") {
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
			granted = GrantsPermission(p, req)
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
