package access

import (
	"errors"
	"fmt"

	"github.com/gin-gonic/gin"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/logging"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/internal/server/providers"
	"github.com/infrahq/infra/uid"
)

// isIdentitySelf is used by authorization checks to see if the calling identity is requesting their own attributes
func isIdentitySelf(c *gin.Context, userID uid.ID) (bool, error) {
	identity := AuthenticatedIdentity(c)
	return identity != nil && identity.ID == userID, nil
}

// AuthenticatedIdentity returns the identity that is associated with the access key
// that was used to authenticate the request.
// Returns nil if there is no identity in the context, which likely means the
// request was not authenticated.
func AuthenticatedIdentity(c *gin.Context) *models.Identity {
	if raw, ok := c.Get("identity"); ok {
		if identity, ok := raw.(*models.Identity); ok {
			return identity
		}
	}
	return nil
}

func GetIdentity(c *gin.Context, id uid.ID) (*models.Identity, error) {
	db, err := hasAuthorization(c, id, isIdentitySelf, models.InfraAdminRole, models.InfraViewRole, models.InfraConnectorRole)
	if err != nil {
		return nil, err
	}

	return data.GetIdentity(db.Preload("Providers"), data.ByID(id))
}

func CreateIdentity(c *gin.Context, identity *models.Identity) error {
	db, err := RequireInfraRole(c, models.InfraAdminRole)
	if err != nil {
		return err
	}

	return data.CreateIdentity(db, identity)
}

func InfraConnectorIdentity(c *gin.Context) *models.Identity {
	return data.InfraConnectorIdentity(getDB(c))
}

// TODO (https://github.com/infrahq/infra/issues/2318) remove provider user, not user.
func DeleteIdentity(c *gin.Context, id uid.ID) error {
	self, err := isIdentitySelf(c, id)
	if err != nil {
		return err
	}

	if self {
		return fmt.Errorf("cannot delete self: %w", internal.ErrForbidden)
	}

	if InfraConnectorIdentity(c).ID == id {
		return internal.ErrForbidden
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

func ListIdentities(c *gin.Context, name string, groupID uid.ID, ids []uid.ID, pg models.Pagination) ([]models.Identity, error) {
	db, err := RequireInfraRole(c, models.InfraAdminRole, models.InfraViewRole, models.InfraConnectorRole)
	if err != nil {
		return nil, err
	}

	selectors := []data.SelectorFunc{
		data.ByOptionalName(name),
		data.ByOptionalIDs(ids),
		data.ByPagination(pg),
	}

	if groupID != 0 {
		return data.ListIdentitiesByGroup(db.Preload("Providers"), groupID, selectors...)
	}

	return data.ListIdentities(db.Preload("Providers"), selectors...)
}

func GetContextProviderIdentity(c *gin.Context) (*models.Provider, string, error) {
	// added by the authentication middleware
	identity := AuthenticatedIdentity(c)
	if identity == nil {
		return nil, "", errors.New("user does not have session with an identity provider")
	}

	// does not need authorization check, this action is limited to the calling user
	db := getDB(c)

	accessKey := currentAccessKey(c)

	providerUser, err := data.GetProviderUser(db, accessKey.ProviderID, identity.ID)
	if err != nil {
		return nil, "", err
	}

	provider, err := data.GetProvider(db, data.ByID(providerUser.ProviderID))
	if err != nil {
		return nil, "", fmt.Errorf("user info provider: %w", err)
	}

	return provider, providerUser.RedirectURL, nil
}

// UpdateIdentityInfoFromProvider calls the identity provider used to authenticate this user session to update their current information
func UpdateIdentityInfoFromProvider(c *gin.Context, oidc providers.OIDC) error {
	// added by the authentication middleware
	identity := AuthenticatedIdentity(c)
	if identity == nil {
		return errors.New("user does not have session with an identity provider")
	}

	// does not need authorization check, this action is limited to the calling user
	db := getDB(c)

	accessKey := currentAccessKey(c)

	provider, err := data.GetProvider(db, data.ByID(accessKey.ProviderID))
	if err != nil {
		return fmt.Errorf("user info provider: %w", err)
	}

	// get current identity provider groups
	err = oidc.SyncProviderUser(db, identity, provider)
	if err != nil {
		if errors.Is(err, internal.ErrBadGateway) {
			return err
		}

		if nestedErr := data.DeleteAccessKeys(db, data.ByIssuedFor(identity.ID)); nestedErr != nil {
			logging.S.Errorf("failed to revoke invalid user session: %s", nestedErr)
		}

		return fmt.Errorf("sync user: %w", err)
	}

	return nil
}
