package access

import (
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/registry/data"
	"github.com/infrahq/infra/internal/registry/models"
	"github.com/infrahq/infra/uid"
)

const (
	PermissionAPIToken       Permission = "infra.apitoken.*"
	PermissionAPITokenCreate Permission = "infra.apitoken.create"
	PermissionAPITokenRead   Permission = "infra.apitoken.read"
	PermissionAPITokenDelete Permission = "infra.apitoken.delete"
)

func ListAPITokens(c *gin.Context) ([]models.APIToken, error) {
	db, err := requireAuthorization(c, PermissionAPITokenRead)
	if err != nil {
		return nil, err
	}

	return data.ListAPITokens(db)
}

func CreateAPIToken(c *gin.Context, token *models.APIToken) (body string, err error) {
	db, err := requireAuthorization(c, PermissionAPITokenCreate)
	if err != nil {
		return "", err
	}

	// do not let a caller create a token with more permissions than they have
	permissions, ok := c.MustGet("permissions").(string)
	if !ok {
		// there should have been permissions set by this point
		return "", internal.ErrForbidden
	}

	if token.Permissions != "" && !AllRequired(strings.Split(permissions, " "), strings.Split(token.Permissions, " ")) {
		return "", fmt.Errorf("cannot create an API token with permission not granted to the token issuer")
	}

	body, err = data.CreateAPIToken(db, token)
	if err != nil {
		return "", fmt.Errorf("create token: %w", err)
	}

	return body, err
}

func DeleteAPIToken(c *gin.Context, id uid.ID) error {
	db, err := requireAuthorization(c, PermissionAPITokenDelete)
	if err != nil {
		return err
	}

	return data.DeleteAPIToken(db, id)
}

func DeleteAllUserAPITokens(c *gin.Context) error {
	user := CurrentUser(c)
	db := getDB(c)

	return data.DeleteAPITokens(db, data.ByUserID(user.ID))
}
