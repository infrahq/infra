package access

import (
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

const (
	PermissionGrant       Permission = "infra.grant.*"
	PermissionGrantCreate Permission = "infra.grant.create"
	PermissionGrantRead   Permission = "infra.grant.read"
	PermissionGrantUpdate Permission = "infra.grant.update"
	PermissionGrantDelete Permission = "infra.grant.delete"
)

func GetGrant(c *gin.Context, id uid.ID) (*models.Grant, error) {
	db, err := requireAuthorization(c, PermissionGrantRead)
	if err != nil {
		return nil, err
	}

	return data.GetGrant(db, data.ByID(id))
}

func ListGrants(c *gin.Context, identity string, resource string, privilege string) ([]models.Grant, error) {
	db, err := requireAuthorization(c, PermissionGrantRead)
	if err != nil {
		return nil, err
	}

	return data.ListGrants(db, data.ByIdentity(identity), data.ByResource(resource), data.ByPrivilege(privilege))
}

func ListUserGrants(c *gin.Context, userID uid.ID) ([]models.Grant, error) {
	db, err := requireAuthorizationWithCheck(c, PermissionGrantRead, func(user *models.User) bool {
		return userID == user.ID
	})
	if err != nil {
		return nil, err
	}

	return data.ListUserGrants(db, userID)
}

func ListGroupGrants(c *gin.Context, groupID uid.ID) ([]models.Grant, error) {
	db, err := requireAuthorization(c, PermissionGrantRead)
	if err != nil {
		return nil, err
	}

	return data.ListGroupGrants(db, groupID)
}

func CreateGrant(c *gin.Context, grant *models.Grant) error {
	db, err := requireAuthorization(c, PermissionGrantCreate)
	if err != nil {
		return err
	}

	// TODO (https://github.com/infrahq/infra/issues/855): replace permissions with actual grant models
	if grant.Resource == "infra" && grant.Privilege == "admin" {
		userID, err := uid.ParseString(strings.TrimPrefix(grant.Identity, "u:"))
		if err != nil {
			return fmt.Errorf("invalid identity id: %w", err)
		}

		user, err := data.GetUser(db, data.ByID(userID))
		if err != nil {
			return fmt.Errorf("could not get user: %w", err)
		}

		user.Permissions = "infra.*"
		err = data.SaveUser(db, user)
		if err != nil {
			return err
		}
	}

	return data.CreateGrant(db, grant)
}

func DeleteGrant(c *gin.Context, id uid.ID) error {
	db, err := requireAuthorization(c, PermissionGrantDelete)
	if err != nil {
		return err
	}

	// TODO: replace permissions with actual grant models
	grant, err := data.GetGrant(db, data.ByID(id))
	if err != nil {
		return err
	}

	if grant.Resource == "infra" && grant.Privilege == "admin" {
		userID, err := uid.ParseString(strings.TrimPrefix(grant.Identity, "u:"))
		if err != nil {
			return fmt.Errorf("invalid identity id: %w", err)
		}

		user, err := data.GetUser(db, data.ByID(userID))
		if err != nil {
			return fmt.Errorf("could not get user: %w", err)
		}

		user.Permissions = DefaultUserPermissions
		err = data.SaveUser(db, user)
		if err != nil {
			return err
		}
	}

	return data.DeleteGrants(db, data.ByID(id))
}
