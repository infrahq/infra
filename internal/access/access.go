package access

import (
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/infrahq/infra/internal"
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

	switch len(bearer) {
	case models.TokenLength:
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

		// token is valid, check if token permissions need to be updated to match parent user
		if token.UserID != uuid.Nil && token.Permissions != token.User.Permissions {
			token.Permissions = token.User.Permissions

			if _, err := data.UpdateToken(db, token); err != nil {
				return fmt.Errorf("update user token permissions: %w", err)
			}
		}

		c.Set("authentication", bearer)
		c.Set("permissions", token.Permissions)

		return nil
	case models.APIKeyLength:
		apiKey, err := data.GetAPIKey(db, &models.APIKey{Key: bearer})
		if err != nil {
			return fmt.Errorf("rejected invalid API key: %w", err)
		}

		c.Set("authentication", bearer)
		c.Set("permissions", apiKey.Permissions)

		return nil
	}

	return fmt.Errorf("rejected token of invalid length")
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
		if RequirePermission(p, string(require)) {
			return db, nil
		}
	}

	return nil, internal.ErrForbidden
}

func RequirePermission(permission, require string) bool {
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
