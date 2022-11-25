package access

import (
	"fmt"

	"github.com/gin-gonic/gin"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

func ListAccessKeys(c *gin.Context, identityID uid.ID, name string, showExpired bool, p *data.Pagination) ([]models.AccessKey, error) {
	rCtx := GetRequestContext(c)
	if identityID == rCtx.Authenticated.User.ID {
		// can list own keys
	} else {
		roles := []string{models.InfraAdminRole, models.InfraViewRole}
		_, err := RequireInfraRole(c, roles...)
		if err != nil {
			return nil, HandleAuthErr(err, "access keys", "list", roles...)
		}
	}

	opts := data.ListAccessKeyOptions{
		Pagination:     p,
		IncludeExpired: showExpired,
		ByIssuedForID:  identityID,
		ByName:         name,
	}
	return data.ListAccessKeys(rCtx.DBTxn, opts)
}

func CreateAccessKey(c *gin.Context, accessKey *models.AccessKey) (string, error) {
	rCtx := GetRequestContext(c)

	if rCtx.Authenticated.AccessKey != nil && !rCtx.Authenticated.AccessKey.Scopes.Includes(models.ScopeAllowCreateAccessKey) {
		if connector := data.InfraConnectorIdentity(rCtx.DBTxn); connector.ID != accessKey.IssuedFor {
			// non-login access keys can not currently create non-connector access keys.
			return "", fmt.Errorf("%w: cannot use an access key to create other access keys", internal.ErrBadRequest)
		}
	}

	err := IsAuthorized(rCtx, models.InfraAdminRole)
	if err != nil && accessKey.IssuedFor != rCtx.Authenticated.User.ID {
		return "", HandleAuthErr(err, "access key", "create", models.InfraAdminRole)
	}

	body, err := data.CreateAccessKey(rCtx.DBTxn, accessKey)
	if err != nil {
		return "", fmt.Errorf("create token: %w", err)
	}

	return body, err
}

func DeleteAccessKey(rCtx RequestContext, id uid.ID, name string) error {
	var key *models.AccessKey
	var err error

	if id != 0 {
		key, err = data.GetAccessKey(rCtx.DBTxn, data.GetAccessKeysOptions{ByID: id})
		if err != nil {
			return err
		}
	} else {
		// if the specific key isn't specified, look up the key by name for the current user
		opts := data.ListAccessKeyOptions{
			IncludeExpired: false,
			ByIssuedForID:  rCtx.Authenticated.User.ID,
			ByName:         name,
		}
		keys, err := data.ListAccessKeys(rCtx.DBTxn, opts)
		if err != nil {
			return err
		}
		if len(keys) > 0 {
			key = &keys[0]
		} else {
			return fmt.Errorf("%w: no key named '%s' found", internal.ErrNotFound, name)
		}
	}

	if key.IssuedFor == rCtx.Authenticated.User.ID {
		// users can delete their own keys
	} else {
		if err := IsAuthorized(rCtx, models.InfraAdminRole); err != nil {
			return HandleAuthErr(err, "access key", "delete", models.InfraAdminRole)
		}
	}

	if rCtx.Authenticated.AccessKey.ID == key.ID {
		return fmt.Errorf("%w: cannot delete the access key used by this request", internal.ErrBadRequest)
	}

	return data.DeleteAccessKeys(rCtx.DBTxn, data.DeleteAccessKeysOptions{ByID: key.ID})
}

func DeleteRequestAccessKey(c RequestContext) error {
	// does not need authorization check, this action is limited to the calling key

	id := c.Authenticated.AccessKey.ID
	return data.DeleteAccessKeys(c.DBTxn, data.DeleteAccessKeysOptions{ByID: id})
}
