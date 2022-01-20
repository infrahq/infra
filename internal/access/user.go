package access

import (
	"github.com/gin-gonic/gin"
	"github.com/infrahq/infra/uuid"

	"github.com/infrahq/infra/internal/registry/data"
	"github.com/infrahq/infra/internal/registry/models"
)

const (
	PermissionUser       Permission = "infra.user.*"
	PermissionUserCreate Permission = "infra.user.create"
	PermissionUserRead   Permission = "infra.user.read"
	PermissionUserUpdate Permission = "infra.user.update"
	PermissionUserDelete Permission = "infra.user.delete"
)

var RoleAdmin = []Permission{PermissionAllInfra}

func currentUser(c *gin.Context) *models.User {
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
func currentUserID(c *gin.Context) (id uuid.UUID, found bool) {
	userIDObj, exists := c.Get("user_id")
	if !exists {
		return 0, false
	}

	userID, ok := userIDObj.(uuid.UUID)
	if !ok {
		return 0, false
	}

	if userID == 0 {
		return 0, false
	}

	return userID, true
}

func GetUser(c *gin.Context, id uuid.UUID) (*models.User, error) {
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
