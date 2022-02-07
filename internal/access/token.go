package access

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
)

const (
	PermissionToken       Permission = "infra.token.*"
	PermissionTokenCreate Permission = "infra.token.create"
)

func CreateUserToken(c *gin.Context) (token *models.Token, err error) {
	user := CurrentUser(c)

	if user == nil {
		return nil, fmt.Errorf("no active user")
	}

	db, err := requireAuthorizationWithCheck(c, PermissionTokenCreate, func(currentUser *models.User) bool {
		return currentUser.ID == user.ID
	})
	if err != nil {
		return nil, err
	}

	return data.CreateToken(db, user.ID)
}
