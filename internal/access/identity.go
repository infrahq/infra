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
	db, err := hasAuthorization(c, id, isIdentitySelf, models.InfraAdminRole, models.InfraViewRole, models.InfraConnectorRole)
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
	self, err := isIdentitySelf(c, id)
	if err != nil {
		return err
	}

	if self {
		return fmt.Errorf("cannot delete self: %w", internal.ErrForbidden)
	}

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

	err = data.DeleteGrants(db, data.BySubject(uid.NewIdentityPolymorphicID(id)))
	if err != nil {
		return fmt.Errorf("delete identity creds: %w", err)
	}

	return data.DeleteIdentity(db, id)
}

func ListIdentities(c *gin.Context, name string, showInactive bool) ([]models.Identity, error) {
	db, err := RequireInfraRole(c, models.InfraAdminRole, models.InfraViewRole, models.InfraConnectorRole)
	if err != nil {
		return nil, err
	}

	identities, err := data.ListIdentities(db, data.ByOptionalName(name))
	if err != nil {
		return nil, err
	}

	if showInactive {
		return identities, nil
	}

	// filter out identities that do not have a linked provider

	providerIdentities, err := data.ListProviderUsers(db)
	if err != nil {
		return nil, err
	}

	// map identity ID of providerIdentity to an identity
	idToIdentity := make(map[uid.ID]*models.Identity)
	for i := range identities {
		idToIdentity[identities[i].ID] = &identities[i]
	}

	var activeIdentities []models.Identity

	for _, active := range providerIdentities {
		activeIdentity := idToIdentity[active.IdentityID]
		if activeIdentity != nil {
			activeIdentities = append(activeIdentities, *activeIdentity)

			// remove from the map so we don't add identities multiple times
			idToIdentity[active.IdentityID] = nil
		}
	}

	return activeIdentities, nil
}

// UpdateUserInfoFromProvider calls the user info endpoint of an external identity provider to see a user's current attributes
func UpdateUserInfoFromProvider(c *gin.Context, info *authn.UserInfo, user *models.Identity, provider *models.Provider) error {
	// no auth, this is not publically exposed
	db := getDB(c)

	// add user to groups they are currently in
	var groups []string

	if info.Groups != nil {
		for i := range *info.Groups {
			name := (*info.Groups)[i]
			groups = append(groups, name)
		}
	}

	if err := data.AssignIdentityToGroups(db, user, provider, groups); err != nil {
		return fmt.Errorf("assign identity to groups: %w", err)
	}

	return nil
}
