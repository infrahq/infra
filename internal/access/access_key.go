package access

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

func currentAccessKey(c *gin.Context) *models.AccessKey {
	accessKey, ok := c.MustGet("key").(*models.AccessKey)
	if !ok {
		return nil
	}

	return accessKey
}

func ListAccessKeys(c *gin.Context, machineID uid.ID, name string) ([]models.AccessKey, error) {
	db, err := RequireInfraRole(c, models.InfraAdminRole)
	if err != nil {
		return nil, err
	}

	return data.ListAccessKeys(db, data.ByMachineIDIssuedFor(machineID), data.ByName(name))
}

func CreateAccessKey(c *gin.Context, accessKey *models.AccessKey, machineID uid.ID) (body string, err error) {
	db, err := RequireInfraRole(c, models.InfraAdminRole)
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
	db, err := RequireInfraRole(c, models.InfraAdminRole)
	if err != nil {
		return err
	}

	return data.DeleteAccessKeys(db, data.ByID(id))
}

func DeleteRequestAccessKey(c *gin.Context) error {
	// does not need authorization check, this action is limited to the calling key
	key := currentAccessKey(c)

	db := getDB(c)

	return data.DeleteAccessKey(db, key.ID)
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
func ExchangeAccessKey(c *gin.Context, requestingAccessKey string, expiry time.Time) (string, *models.Machine, error) {
	db := getDB(c)

	validatedRequestKey, err := data.ValidateAccessKey(db, requestingAccessKey)
	if err != nil {
		return "", nil, fmt.Errorf("%w: invalid access key in exchange: %v", internal.ErrUnauthorized, err)
	}

	if expiry.After(validatedRequestKey.ExpiresAt) {
		return "", nil, fmt.Errorf("%w: cannot exchange an access key for another access key with a longer lifetime", internal.ErrBadRequest)
	}

	if !validatedRequestKey.IssuedFor.IsMachine() {
		// this flow isn't supported for user identities yet
		return "", nil, internal.ErrNotImplemented
	}

	machineID, err := validatedRequestKey.IssuedFor.ID()
	if err != nil {
		return "", nil, fmt.Errorf("%w: parse exchange issue id: %v", internal.ErrUnauthorized, err)
	}

	machine, err := data.GetMachine(db, data.ByID(machineID))
	if err != nil {
		return "", nil, fmt.Errorf("get machine exchange: %w", err)
	}

	exchangedAccessKey := &models.AccessKey{
		IssuedFor: validatedRequestKey.IssuedFor,
		ExpiresAt: expiry,
	}

	secret, err := data.CreateAccessKey(db, exchangedAccessKey)
	if err != nil {
		return "", nil, fmt.Errorf("create exchanged token: %w", err)
	}

	return secret, machine, nil
}
