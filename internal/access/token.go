package access

import (
	"fmt"

	"github.com/gin-gonic/gin"

	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
)

func CreateUserToken(c *gin.Context) (token *models.Token, err error) {
	user := CurrentUser(c)
	if user == nil {
		return nil, fmt.Errorf("no active user")
	}

	// does not need authorization check, limited to calling user
	db := getDB(c)

	return data.CreateUserToken(db, user.ID)
}

func CreateMachineToken(c *gin.Context) (token *models.Token, err error) {
	machine := CurrentMachine(c)

	if machine == nil {
		return nil, fmt.Errorf("no active machine")
	}

	// does not need authorization check, limited to calling machine
	db := getDB(c)

	return data.CreateMachineToken(db, machine.ID)
}
