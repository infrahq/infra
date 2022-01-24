package access

import (
	"errors"
	"fmt"

	"github.com/gin-gonic/gin"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/registry/authn"
	"github.com/infrahq/infra/internal/registry/data"
	"github.com/infrahq/infra/internal/registry/models"
	"github.com/infrahq/infra/uid"
)

const (
	PermissionUser       Permission = "infra.user.*"
	PermissionUserCreate Permission = "infra.user.create"
	PermissionUserRead   Permission = "infra.user.read"
	PermissionUserUpdate Permission = "infra.user.update"
	PermissionUserDelete Permission = "infra.user.delete"
)

var RoleAdmin = []Permission{PermissionAllInfra}

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

// nolint until this is used
// nolint
func currentUserID(c *gin.Context) (id uid.ID, found bool) {
	userIDObj, exists := c.Get("user_id")
	if !exists {
		return 0, false
	}

	userID, ok := userIDObj.(uid.ID)
	if !ok {
		return 0, false
	}

	if userID == 0 {
		return 0, false
	}

	return userID, true
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

	return data.GetUser(data.UserAssociations(db), data.ByID(id))
}

func ListUsers(c *gin.Context, email string) ([]models.User, error) {
	db, err := requireAuthorizationWithCheck(c, PermissionUserRead, func(currentUser *models.User) bool {
		return currentUser.Email == email
	})
	if err != nil {
		return nil, err
	}

	return data.ListUsers(data.UserAssociations(db), data.ByEmail(email))
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

			if group, err = data.CreateGroup(db, &models.Group{Name: name}); err != nil {
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
