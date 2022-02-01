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
	PermissionAccessKey       Permission = "infra.accesskey.*"
	PermissionAccessKeyCreate Permission = "infra.accesskey.create"
	PermissionAccessKeyRead   Permission = "infra.accesskey.read"
	PermissionAccessKeyDelete Permission = "infra.accesskey.delete"
)

func ListAccessKeys(c *gin.Context) ([]models.AccessKey, error) {
	db, err := requireAuthorization(c, PermissionAccessKeyRead)
	if err != nil {
		return nil, err
	}

	return data.ListAccessKeys(db)
}

func CreateAccessKey(c *gin.Context, token *models.AccessKey) (body string, err error) {
	db, err := requireAuthorization(c, PermissionAccessKeyCreate)
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
		return "", fmt.Errorf("cannot create an access key with permission not granted to the token issuer")
	}

	body, err = data.CreateAccessKey(db, token)
	if err != nil {
		return "", fmt.Errorf("create token: %w", err)
	}

	return body, err
}

func DeleteAccessKey(c *gin.Context, id uid.ID) error {
	db, err := requireAuthorization(c, PermissionAccessKeyDelete)
	if err != nil {
		return err
	}

	return data.DeleteAccessKey(db, id)
}

func DeleteAllUserAccessKeys(c *gin.Context) error {
	user := CurrentUser(c)
	db := getDB(c)

	return data.DeleteAccessKeys(db, data.ByUserID(user.ID))
}
