package access

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

func ListAccessKeys(c *gin.Context, machineID uid.ID, name string) ([]models.AccessKey, error) {
	db, err := requireInfraRole(c, AdminRole)
	if err != nil {
		return nil, err
	}

	return data.ListAccessKeys(db, data.ByMachineIDIssuedFor(machineID), data.ByName(name))
}

func CreateAccessKey(c *gin.Context, accessKey *models.AccessKey, machineID uid.ID) (body string, err error) {
	db, err := requireInfraRole(c, AdminRole)
	if err != nil {
		return "", err
	}

	_, err = data.GetMachine(db, data.ByID(machineID))
	if err != nil {
		return "", fmt.Errorf("get access key machine: %w", err)
	}

	body, err = data.CreateAccessKey(db, accessKey)
	if err != nil {
		return "", fmt.Errorf("create token: %w", err)
	}

	return body, err
}

func DeleteAccessKey(c *gin.Context, id uid.ID) error {
	db, err := requireInfraRole(c, AdminRole)
	if err != nil {
		return err
	}

	return data.DeleteAccessKeys(db, data.ByID(id))
}

func DeleteAllUserAccessKeys(c *gin.Context) error {
	// does not need authorization check, this action is limited to the calling user
	user := CurrentUser(c)
	if user == nil {
		return fmt.Errorf("no active user")
	}

	db := getDB(c)

	return data.DeleteAccessKeys(db, data.ByUserIDIssuedFor(user.ID))
}
