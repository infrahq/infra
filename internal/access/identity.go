package access

import (
	"errors"
	"fmt"

	"github.com/gin-gonic/gin"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/logging"
	"github.com/infrahq/infra/internal/server/authn"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

// isIdentitySelf is used by authorization checks to see if the calling identity is requesting their own attributes
func isIdentitySelf(c *gin.Context, requestedResourceID uid.ID) (bool, error) {
	identity := AuthenticatedIdentity(c)
	return identity != nil && identity.ID == requestedResourceID, nil
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

	return data.GetIdentity(db, data.ByID(id))
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

func ListIdentities(c *gin.Context, name string, ids []uid.ID, pg *models.Pagination) ([]models.Identity, error) {
	db, err := RequireInfraRole(c, models.InfraAdminRole, models.InfraViewRole, models.InfraConnectorRole)
	if err != nil {
		return nil, err
	}

	return data.ListIdentities(db, data.ByOptionalName(name), data.ByOptionalIDs(ids), data.ByPagination(pg))
}

// UpdateUserInfoFromProvider calls the user info endpoint of an external identity provider to see a user's current attributes
func UpdateUserInfoFromProvider(c *gin.Context, info *authn.InfoClaims, user *models.Identity, provider *models.Provider) error {
	// no auth, this is not publically exposed
	db := getDB(c)

	// add user to groups they are currently in
	var groups []string

	for i := range info.Groups {
		name := info.Groups[i]
		groups = append(groups, name)
	}

	logging.S.Debugf("%s user authenticated with %q groups", provider.Name, groups)

	if err := data.AssignIdentityToGroups(db, user, provider, groups); err != nil {
		return fmt.Errorf("assign identity to groups: %w", err)
	}

	return nil
}
