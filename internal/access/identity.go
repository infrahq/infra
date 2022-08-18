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
//
// TODO: remove once all callers use RequestContext.Authenticated.User
// See https://github.com/infrahq/infra/issues/2796
func AuthenticatedIdentity(c *gin.Context) *models.Identity {
	if raw, ok := c.Get("identity"); ok {
		if identity, ok := raw.(*models.Identity); ok {
			return identity
		}
	}
	return nil
}

func GetIdentity(c *gin.Context, id uid.ID) (*models.Identity, error) {
	roles := []string{models.InfraAdminRole, models.InfraViewRole, models.InfraConnectorRole}
	db, err := hasAuthorization(c, id, isIdentitySelf, roles...)
	if err != nil {
		return nil, HandleAuthErr(err, "user", "get", roles...)
	}

	return data.GetIdentity(db.Preload("Providers"), data.ByID(id))
}

func CreateIdentity(c *gin.Context, identity *models.Identity) error {
	db, err := RequireInfraRole(c, models.InfraAdminRole)
	if err != nil {
		return HandleAuthErr(err, "user", "create", models.InfraAdminRole)
	}

	return data.CreateIdentity(db, identity)
}

func InfraConnectorIdentity(c *gin.Context) *models.Identity {
	return data.InfraConnectorIdentity(getDB(c))
}

// This checks if the delete request is allowed
func CanDeleteIdentity(c *gin.Context, id uid.ID) error {
	self, err := isIdentitySelf(c, id)
	if err != nil {
		return err
	}
	if self {
		return fmt.Errorf("cannot delete self: %w", internal.ErrBadRequest)
	}

	if InfraConnectorIdentity(c).ID == id {
		return fmt.Errorf("%w: the connector user can not be deleted", internal.ErrBadRequest)
	}

	return nil
}

func DeleteIdentity2(c *gin.Context, id uid.ID) error {
	if err := CanDeleteIdentity(c, id); err != nil {
		return err
	}

	db, err := RequireInfraRole(c, models.InfraAdminRole)
	if err != nil {
		return HandleAuthErr(err, "user", "delete", models.InfraAdminRole)
	}

	return data.DeleteIdentity(db, id)
}

func DeleteCredentialForUser(c *gin.Context, userID uid.ID) error {
	db, err := RequireInfraRole(c, models.InfraAdminRole)
	if err != nil {
		return HandleAuthErr(err, "user", "delete", models.InfraAdminRole)
	}

	credential, err := data.GetCredential(db, data.ByIdentityID(userID))
	if err != nil && !errors.Is(err, internal.ErrNotFound) {
		return fmt.Errorf("get delete identity creds: %w", err)
	}

	if credential != nil {
		err := data.DeleteCredential(db, credential.ID)
		if err != nil {
			return fmt.Errorf("delete identity creds: %w", err)
		}
	}
	return nil
}

// TODO (https://github.com/infrahq/infra/issues/2318) remove provider user, not user.
func DeleteIdentity(c *gin.Context, id uid.ID) error {
	self, err := isIdentitySelf(c, id)
	if err != nil {
		return err
	}

	if self {
		return fmt.Errorf("cannot delete self: %w", internal.ErrBadRequest)
	}

	if InfraConnectorIdentity(c).ID == id {
		return fmt.Errorf("%w: the connector user can not be deleted", internal.ErrBadRequest)
	}

	db, err := RequireInfraRole(c, models.InfraAdminRole)
	if err != nil {
		return HandleAuthErr(err, "user", "delete", models.InfraAdminRole)
	}

	if err := data.DeleteAccessKeys(db, data.ByIssuedFor(id)); err != nil {
		return fmt.Errorf("delete identity access keys: %w", err)
	}

	groups, err := data.ListGroups(db, nil, data.ByGroupMember(id))
	if err != nil {
		return fmt.Errorf("list groups for identity: %w", err)
	}
	for _, group := range groups {
		err = data.RemoveUsersFromGroup(db, group.ID, []uid.ID{id})
		if err != nil {
			return fmt.Errorf("delete group membership for identity: %w", err)
		}
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

func ListIdentities(c *gin.Context, name string, groupID uid.ID, ids []uid.ID, showSystem bool, p *models.Pagination) ([]models.Identity, error) {
	roles := []string{models.InfraAdminRole, models.InfraViewRole, models.InfraConnectorRole}
	db, err := RequireInfraRole(c, roles...)
	if err != nil {
		return nil, HandleAuthErr(err, "users", "list", roles...)
	}

	selectors := []data.SelectorFunc{
		data.ByOptionalName(name),
		data.ByOptionalIDs(ids),
		data.ByOptionalIdentityGroupID(groupID),
	}

	if !showSystem {
		selectors = append(selectors, data.NotName(models.InternalInfraConnectorIdentityName))
	}

	return data.ListIdentities(db.Preload("Providers"), p, selectors...)
}

func GetContextProviderIdentity(c RequestContext) (*models.Provider, string, error) {
	// does not need authorization check, this action is limited to the calling user
	identity := c.Authenticated.User
	if identity == nil {
		return nil, "", errors.New("user does not have session with an identity provider")
	}

	providerUser, err := data.GetProviderUser(c.DBTxn, c.Authenticated.AccessKey.ProviderID, identity.ID)
	if err != nil {
		return nil, "", err
	}

	provider, err := data.GetProvider(c.DBTxn, data.ByID(providerUser.ProviderID))
	if err != nil {
		return nil, "", fmt.Errorf("user info provider: %w", err)
	}

	return provider, providerUser.RedirectURL, nil
}

// UpdateIdentityInfoFromProvider calls the identity provider used to authenticate this user session to update their current information
func UpdateIdentityInfoFromProvider(c RequestContext, oidc providers.OIDCClient) error {
	// does not need authorization check, this action is limited to the calling user
	ctx := c.Request.Context()

	// added by the authentication middleware
	identity := c.Authenticated.User
	if identity == nil {
		return errors.New("user does not have session with an identity provider")
	}

	db := c.DBTxn
	provider, err := data.GetProvider(db, data.ByID(c.Authenticated.AccessKey.ProviderID))
	if err != nil {
		return fmt.Errorf("user info provider: %w", err)
	}

	// get current identity provider groups and account status
	err = data.SyncProviderUser(ctx, db, identity, provider, oidc)
	if err != nil {
		if errors.Is(err, internal.ErrBadGateway) {
			return err
		}

		if nestedErr := data.DeleteAccessKeys(db, data.ByIssuedFor(identity.ID)); nestedErr != nil {
			logging.Errorf("failed to revoke invalid user session: %s", nestedErr)
		}

		if nestedErr := data.DeleteProviderUsers(db, data.ByIdentityID(identity.ID), data.ByProviderID(provider.ID)); nestedErr != nil {
			logging.Errorf("failed to delete provider user: %s", nestedErr)
		}

		return fmt.Errorf("sync user: %w", err)
	}

	return nil
}
