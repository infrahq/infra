package access

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/logging"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

func ListAccessKeys(c *gin.Context, machineID uid.ID, name string) ([]models.AccessKey, error) {
	db, err := requireInfraRole(c, models.InfraAdminRole)
	if err != nil {
		return nil, err
	}

	return data.ListAccessKeys(db, data.ByMachineIDIssuedFor(machineID), data.ByName(name))
}

func CreateAccessKey(c *gin.Context, accessKey *models.AccessKey, machineID uid.ID) (body string, err error) {
	db, err := requireInfraRole(c, models.InfraAdminRole)
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
	db, err := requireInfraRole(c, models.InfraAdminRole)
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

// ExchangeAccessKey allows a key exchange to get a new key with a shorter lifetime
func ExchangeAccessKey(c *gin.Context, requestingAccessKey string, expiry time.Time) (identity *uid.PolymorphicID, name, secret string, err error) {
	db := getDB(c)

	validatedRequestKey, err := data.ValidateAccessKey(db, requestingAccessKey)
	if err != nil {
		logging.S.Debugf("access key was found to be invalid in exchange: %s", err)
		return nil, "", "", fmt.Errorf("unauthorized")
	}

	if expiry.After(validatedRequestKey.ExpiresAt) {
		return nil, "", "", fmt.Errorf("cannot exchange an access key for another access key with a longer lifetime")
	}

	if !validatedRequestKey.IssuedFor.IsMachine() {
		// this flow isn't supported for user identities yet
		return nil, "", "", internal.ErrNotImplemented
	}

	machineID, err := validatedRequestKey.IssuedFor.ID()
	if err != nil {
		return nil, "", "", fmt.Errorf("parse exchange issue id: %w", err)
	}

	machine, err := data.GetMachine(db, data.ByID(machineID))
	if err != nil {
		return nil, "", "", fmt.Errorf("get machine exchange: %w", err)
	}

	exchangedAccessKey := &models.AccessKey{
		IssuedFor: validatedRequestKey.IssuedFor,
		ExpiresAt: expiry,
	}

	secret, err = data.CreateAccessKey(db, exchangedAccessKey)
	if err != nil {
		return nil, "", "", fmt.Errorf("create exchanged token: %w", err)
	}

	return &validatedRequestKey.IssuedFor, machine.Name, secret, nil
}
