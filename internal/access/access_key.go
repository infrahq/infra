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

func ListAccessKeys(c *gin.Context, identityID uid.ID, name string) ([]models.AccessKey, error) {
	db, err := RequireInfraRole(c, models.InfraAdminRole, models.InfraViewRole)
	if err != nil {
		return nil, err
	}

	return data.ListAccessKeys(db, data.ByOptionalIssuedFor(identityID), data.ByOptionalName(name))
}

func CreateAccessKey(c *gin.Context, accessKey *models.AccessKey, identityID uid.ID) (body string, err error) {
	db, err := RequireInfraRole(c, models.InfraAdminRole)
	if err != nil {
		return "", err
	}

	identity, err := data.GetIdentity(db, data.ByID(identityID))
	if err != nil {
		return "", fmt.Errorf("get access key identity: %w", err)
	}

	if identity.Kind != models.MachineKind {
		// direct access key creation isn't supported for user identities yet
		return "", internal.ErrNotImplemented
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

func DeleteAllIdentityAccessKeys(c *gin.Context) error {
	// does not need authorization check, this action is limited to the calling user
	identity := CurrentIdentity(c)
	if identity == nil {
		return fmt.Errorf("no active identity")
	}

	db := getDB(c)

	return data.DeleteAccessKeys(db, data.ByIssuedFor(identity.ID))
}

// ExchangeAccessKey allows a key exchange to get a new key with a shorter lifetime
func ExchangeAccessKey(c *gin.Context, requestingAccessKey string, expiry time.Time) (string, *models.Identity, error) {
	db := getDB(c)

	validatedRequestKey, err := data.ValidateAccessKey(db, requestingAccessKey)
	if err != nil {
		return "", nil, fmt.Errorf("%w: invalid access key in exchange: %v", internal.ErrUnauthorized, err)
	}

	if expiry.After(validatedRequestKey.ExpiresAt) {
		return "", nil, fmt.Errorf("%w: cannot exchange an access key for another access key with a longer lifetime", internal.ErrBadRequest)
	}

	identity, err := data.GetIdentity(db, data.ByID(validatedRequestKey.IssuedFor))
	if err != nil {
		return "", nil, fmt.Errorf("get identity exchange: %w", err)
	}

	if identity.Kind != models.MachineKind {
		// this flow isn't supported for user identities yet
		return "", nil, internal.ErrNotImplemented
	}

	exchangedAccessKey := &models.AccessKey{
		IssuedFor:  validatedRequestKey.IssuedFor,
		ProviderID: validatedRequestKey.ProviderID,
		ExpiresAt:  expiry,
	}

	secret, err := data.CreateAccessKey(db, exchangedAccessKey)
	if err != nil {
		return "", nil, fmt.Errorf("create exchanged token: %w", err)
	}

	return secret, identity, nil
}
