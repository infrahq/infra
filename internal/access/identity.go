package access

import (
	"errors"
	"fmt"

	"github.com/gin-gonic/gin"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/server/authn"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

// isIdentitySelf is used by authorization checks to see if the calling identity is requesting their own attributes
func isIdentitySelf(c *gin.Context, requestedResourceID uid.ID) (bool, error) {
	identity := CurrentIdentity(c)
	return identity != nil && identity.ID == requestedResourceID, nil
}

func CurrentIdentity(c *gin.Context) *models.Identity {
	identity, ok := c.MustGet("identity").(*models.Identity)
	if !ok {
		return nil
	}

	return identity
}

func GetIdentity(c *gin.Context, id uid.ID) (*models.Identity, error) {
	db, err := hasAuthorization(c, id, isIdentitySelf, models.InfraAdminRole, models.InfraConnectorRole)
	if err != nil {
		return nil, err
	}

	return data.GetIdentity(db, data.ByID(id))
}

func CreateIdentity(c *gin.Context, identity *models.Identity) error {
	db, err := RequireInfraRole(c, models.InfraAdminRole)
	if err != nil {
		return err
	}

	return data.CreateIdentity(db, identity)
}

func DeleteIdentity(c *gin.Context, id uid.ID) error {
	db, err := RequireInfraRole(c, models.InfraAdminRole)
	if err != nil {
		return err
	}

	if err := data.DeleteAccessKeys(db, data.ByIssuedFor(id)); err != nil {
		return fmt.Errorf("delete identity access keys: %w", err)
	}

	// if an identity does not have credentials in the Infra provider this won't be found, but we can proceed
	credential, err := data.GetCredential(db, data.ByIdentityID(id))
	if err != nil && !errors.Is(err, internal.ErrNotFound) {
		return fmt.Errorf("get delete identity creds: %w", err)
	}

	if credential != nil {
		err := data.DeleteCredential(db, credential.ID)
		if err != nil {
			return fmt.Errorf("delete identity creds: %w", err)
		}
	}

	return data.DeleteIdentity(db, id)
}

func ListIdentities(c *gin.Context, email string, providerID uid.ID) ([]models.Identity, error) {
	db, err := RequireInfraRole(c, models.InfraAdminRole, models.InfraConnectorRole)
	if err != nil {
		return nil, err
	}

	return data.ListIdentities(db, data.ByName(email), data.ByProviderID(providerID))
}

// UpdateUserInfo calls the user info endpoint of an external identity provider to see a user's current attributes
func UpdateUserInfo(c *gin.Context, info *authn.UserInfo, user *models.Identity, provider *models.Provider) error {
	// no auth, this is not publically exposed
	db := getDB(c)

	// add user to groups they are currently in
	var groups []models.Group

	if info.Groups != nil {
		for i := range *info.Groups {
			name := (*info.Groups)[i]

			group, err := data.GetGroup(db, data.ByName(name))
			if err != nil {
				if !errors.Is(err, internal.ErrNotFound) {
					return fmt.Errorf("get group: %w", err)
				}

				group = &models.Group{Name: name}

				if err = data.CreateGroup(db, group); err != nil {
					return fmt.Errorf("create group: %w", err)
				}
			}

			groups = append(groups, *group)
		}
	}

	// remove user from groups they are no longer in
	return data.BindIdentityGroups(db, user, groups...)
}
