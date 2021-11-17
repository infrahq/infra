package access

import (
	"github.com/gin-gonic/gin"

	"github.com/infrahq/infra/internal/data"
)

const (
	PermissionUser       Permission = "infra.user.*"
	PermissionUserCreate Permission = "infra.user.create"
	PermissionUserRead   Permission = "infra.user.read"
	PermissionUserUpdate Permission = "infra.user.update"
	PermissionUserDelete Permission = "infra.user.delete"
)

func GetUser(c *gin.Context, id string) (*data.User, error) {
	db, _, err := RequireAuthorization(c, PermissionUserRead)
	if err != nil {
		return nil, err
	}

	user, err := data.NewUser(id)
	if err != nil {
		return nil, err
	}

	return data.GetUser(data.UserAssociations(db), user)
}

func ListUsers(c *gin.Context, email string) ([]data.User, error) {
	db, _, err := RequireAuthorization(c, PermissionUserRead)
	if err != nil {
		return nil, err
	}

	return data.ListUsers(data.UserAssociations(db), &data.User{Email: email})
}
