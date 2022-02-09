package access

import (
	"errors"
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/server/authn"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

const (
	PermissionUser       Permission = "infra.user.*"
	PermissionUserCreate Permission = "infra.user.create"
	PermissionUserRead   Permission = "infra.user.read"
	PermissionUserUpdate Permission = "infra.user.update"
	PermissionUserDelete Permission = "infra.user.delete"
)

func CurrentUser(c *gin.Context) *models.User {
	userObj, exists := c.Get("user")
	if !exists {
		return nil
	}

	user, ok := userObj.(*models.User)
	if !ok {
		return nil
	}

	return user
}

func GetUser(c *gin.Context, id uid.ID) (*models.User, error) {
	db, err := requireAuthorizationWithCheck(c, PermissionUserRead, func(currentUser *models.User) bool {
		// current user is allowed to fetch their own record,
		// even without the infra.users.read permission
		return currentUser.ID == id
	})
	if err != nil {
		return nil, err
	}

	return data.GetUser(db, data.ByID(id))
}

func CreateUser(c *gin.Context, user *models.User) error {
	db, err := requireAuthorization(c, PermissionUserCreate)
	if err != nil {
		return err
	}

	if user.Permissions == "" {
		user.Permissions = DefaultUserPermissions
	}

	// do not let a caller create a token with more permissions than they have
	permissions, ok := c.MustGet("permissions").(string)
	if !ok {
		// there should have been permissions set by this point
		return internal.ErrForbidden
	}

	if user.Permissions != "" && !AllRequired(strings.Split(permissions, " "), strings.Split(user.Permissions, " ")) {
		return fmt.Errorf("cannot create a user with permission not granted to the user")
	}

	return data.CreateUser(db, user)
}

func ListUsers(c *gin.Context, email string, providerID uid.ID) ([]models.User, error) {
	db, err := requireAuthorizationWithCheck(c, PermissionUserRead, func(currentUser *models.User) bool {
		return currentUser.Email == email
	})
	if err != nil {
		return nil, err
	}

	return data.ListUsers(db, data.ByEmail(email), data.ByProviderID(providerID))
}

func UpdateUserInfo(c *gin.Context, info *authn.UserInfo, user *models.User, provider *models.Provider) error {
	db, err := requireAuthorization(c)
	if err != nil {
		return err
	}

	// add user to groups they are currently in
	var groups []models.Group

	for _, name := range info.Groups {
		group, err := data.GetGroup(db, data.ByName(name))
		if err != nil {
			if !errors.Is(err, internal.ErrNotFound) {
				return fmt.Errorf("get group: %w", err)
			}

			group = &models.Group{Name: name}

			if err = data.CreateGroup(db, group); err != nil {
				return fmt.Errorf("create group: %w", err)
			}
		}

		err = data.AppendProviderGroups(db, provider, group)
		if err != nil {
			return fmt.Errorf("user provider info: %w", err)
		}

		groups = append(groups, *group)
	}

	// remove user from groups they are no longer in
	return data.BindUserGroups(db, user, groups...)
}
