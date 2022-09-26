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

func CreateAccessKey(c *gin.Context, accessKey *models.AccessKey) (body string, err error) {
	rCtx := GetRequestContext(c)

	if rCtx.Authenticated.AccessKey != nil && !rCtx.Authenticated.AccessKey.Scopes.Includes(models.ScopeAllowCreateAccessKey) {
		// non-login access keys can not currently create other access keys.
		return "", fmt.Errorf("%w: cannot use an access key not issued from login to create other access keys", internal.ErrBadRequest)
	}

	if accessKey.IssuedFor == rCtx.Authenticated.User.ID {
		// can create access keys for yourself.
	} else {
		_, err = RequireInfraRole(c, models.InfraAdminRole)
		if err != nil {
			return "", HandleAuthErr(err, "access key", "create", models.InfraAdminRole)
		}
	}

	if accessKey.ProviderID == 0 {
		accessKey.ProviderID = data.InfraProvider(rCtx.DBTxn).ID
	}

	body, err = data.CreateAccessKey(rCtx.DBTxn, accessKey)
	if err != nil {
		return "", fmt.Errorf("create token: %w", err)
	}

	return body, err
}

func DeleteAccessKey(c *gin.Context, id uid.ID, name string) error {
	rCtx := GetRequestContext(c)

	key, err := data.GetAccessKey(rCtx.DBTxn, data.GetAccessKeysOptions{ByID: id, ByName: name})
	if err != nil {
		return err
	}

	if key.IssuedFor == rCtx.Authenticated.User.ID {
		// users can delete their own keys
	} else {
		_, err := RequireInfraRole(c, models.InfraAdminRole)
		if err != nil {
			return HandleAuthErr(err, "access key", "delete", models.InfraAdminRole)
		}
	}

	return data.DeleteAccessKeys(rCtx.DBTxn, data.DeleteAccessKeysOptions{ByID: key.ID})
}

func DeleteRequestAccessKey(c RequestContext) error {
	// does not need authorization check, this action is limited to the calling key

	id := c.Authenticated.AccessKey.ID
	return data.DeleteAccessKeys(c.DBTxn, data.DeleteAccessKeysOptions{ByID: id})
}
