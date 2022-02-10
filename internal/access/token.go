package access

import (
	"fmt"

	"github.com/gin-gonic/gin"

	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
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

	db, err := requireAuthorizationWithCheck(c, PermissionTokenCreate, func(id uid.ID) bool {
		return user.ID == id
	})
	if err != nil {
		return nil, err
	}

	return data.CreateUserToken(db, user.ID)
}

func CreateMachineToken(c *gin.Context) (token *models.Token, err error) {
	machine := CurrentMachine(c)

	if machine == nil {
		return nil, fmt.Errorf("no active machine")
	}

	db, err := requireAuthorizationWithCheck(c, PermissionTokenCreate, func(id uid.ID) bool {
		return machine.ID == id
	})
	if err != nil {
		return nil, err
	}

	return data.CreateMachineToken(db, machine.ID)
}
