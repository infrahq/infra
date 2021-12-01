package access

import (
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/data"
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

func RequireAuthorization(c *gin.Context, require Permission) (*gorm.DB, string, error) {
	val, ok := c.Get("db")
	if !ok {
		return nil, "", fmt.Errorf("database not found")
	}

	db, ok := val.(*gorm.DB)
	if !ok {
		return nil, "", fmt.Errorf("database not found")
	}

	if len(require) == 0 {
		return db, "", nil
	}

	authorization := c.GetString("authorization")
	if authorization == "" {
		return nil, "", internal.ErrInvalid
	}

	switch len(authorization) {
	case data.TokenLength:
		token, err := data.GetToken(db, &data.Token{Key: authorization[:data.TokenKeyLength]})
		if err != nil {
			return nil, "", internal.ErrInvalid
		}

		if err := data.CheckTokenExpired(token); err != nil {
			return nil, "", internal.ErrExpired
		}

		if err := data.CheckTokenSecret(token, authorization); err != nil {
			return nil, "", internal.ErrInvalid
		}

		for _, p := range strings.Split(token.Permissions, " ") {
			if RequirePermission(p, string(require)) {
				return db, authorization, nil
			}
		}

	case data.APIKeyLength:
		apiKey, err := data.GetAPIKey(db, &data.APIKey{Key: authorization})
		if err != nil {
			return nil, "", internal.ErrInvalid
		}

		for _, p := range strings.Split(apiKey.Permissions, " ") {
			if RequirePermission(p, string(require)) {
				return db, authorization, nil
			}
		}
	}

	return nil, "", internal.ErrForbidden
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
