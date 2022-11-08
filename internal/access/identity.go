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
func isIdentitySelf(rCtx RequestContext, opts data.GetIdentityOptions) bool {
	identity := rCtx.Authenticated.User

	if identity == nil {
		return false
	}

	switch {
	case opts.ByID != 0:
		return identity.ID == opts.ByID
	case opts.ByName != "":
		return identity.Name == opts.ByName
	}

	return false
}

func GetIdentity(c *gin.Context, opts data.GetIdentityOptions) (*models.Identity, error) {
	rCtx := GetRequestContext(c)
	// anyone can get their own user data
	if !isIdentitySelf(rCtx, opts) {
		roles := []string{models.InfraAdminRole, models.InfraViewRole, models.InfraConnectorRole}
		err := IsAuthorized(rCtx, roles...)
		if err != nil {
			return nil, HandleAuthErr(err, "user", "get", roles...)
		}
	}

	return data.GetIdentity(rCtx.DBTxn, opts)
}

func CreateIdentity(c *gin.Context, identity *models.Identity) error {
	db, err := RequireInfraRole(c, models.InfraAdminRole)
	if err != nil {
		return HandleAuthErr(err, "user", "create", models.InfraAdminRole)
	}

	return data.CreateIdentity(db, identity)
}

func DeleteIdentity(c *gin.Context, id uid.ID) error {
	rCtx := GetRequestContext(c)
	if isIdentitySelf(rCtx, data.GetIdentityOptions{ByID: id}) {
		return fmt.Errorf("cannot delete self: %w", internal.ErrBadRequest)
	}

	if data.InfraConnectorIdentity(rCtx.DBTxn).ID == id {
		return fmt.Errorf("%w: the connector user can not be deleted", internal.ErrBadRequest)
	}

	db, err := RequireInfraRole(c, models.InfraAdminRole)
	if err != nil {
		return HandleAuthErr(err, "user", "delete", models.InfraAdminRole)
	}

	opts := data.DeleteIdentitiesOptions{
		ByProviderID: data.InfraProvider(db).ID,
		ByID:         id,
	}
	return data.DeleteIdentities(db, opts)
}

func ListIdentities(c *gin.Context, name string, groupID uid.ID, ids []uid.ID, showSystem bool, p *data.Pagination) ([]models.Identity, error) {
	roles := []string{models.InfraAdminRole, models.InfraViewRole, models.InfraConnectorRole}
	db, err := RequireInfraRole(c, roles...)
	if err != nil {
		return nil, HandleAuthErr(err, "users", "list", roles...)
	}

	opts := data.ListIdentityOptions{
		Pagination:    p,
		ByName:        name,
		ByIDs:         ids,
		ByGroupID:     groupID,
		LoadProviders: true,
	}

	if !showSystem {
		opts.ByNotName = models.InternalInfraConnectorIdentityName
	}

	return data.ListIdentities(db, opts)
}

func GetContextProviderIdentity(c RequestContext) (*models.Provider, string, error) {
	// does not need authorization check, this action is limited to the calling user
	provider, err := data.GetProvider(c.DBTxn, data.GetProviderOptions{
		ByID: c.Authenticated.AccessKey.ProviderID,
	})
	if err != nil {
		return nil, "", fmt.Errorf("user info provider: %w", err)
	}

	if provider.Kind == models.ProviderKindInfra {
		// no external verification needed
		logging.L.Trace().Msg("skipped verifying identity within infra provider, not required")
		return provider, "", nil
	}

	identity := c.Authenticated.User
	if identity == nil {
		return nil, "", errors.New("user does not have session with an identity provider")
	}

	providerUser, err := data.GetProviderUser(c.DBTxn, c.Authenticated.AccessKey.ProviderID, identity.ID)
	if err != nil {
		return nil, "", err
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
	provider, err := data.GetProvider(db, data.GetProviderOptions{
		ByID: c.Authenticated.AccessKey.ProviderID,
	})
	if err != nil {
		return fmt.Errorf("user info provider: %w", err)
	}

	// get current identity provider groups and account status
	err = data.SyncProviderUser(ctx, db, identity, provider, oidc)
	if err != nil {
		if errors.Is(err, internal.ErrBadGateway) {
			return err
		}

		if nestedErr := data.DeleteAccessKeys(db, data.DeleteAccessKeysOptions{ByIssuedForID: identity.ID}); nestedErr != nil {
			logging.Errorf("failed to revoke invalid user session: %s", nestedErr)
		}

		if nestedErr := data.DeleteProviderUsers(db, data.DeleteProviderUsersOptions{ByIdentityID: identity.ID, ByProviderID: provider.ID}); nestedErr != nil {
			logging.Errorf("failed to delete provider user: %s", nestedErr)
		}

		return fmt.Errorf("sync user: %w", err)
	}

	return nil
}
