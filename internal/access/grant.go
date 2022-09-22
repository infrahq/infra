package access

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/infrahq/infra/internal/logging"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

func GetGrant(c *gin.Context, id uid.ID) (*models.Grant, error) {
	db, err := RequireInfraRole(c, models.InfraAdminRole)
	if err != nil {
		return nil, HandleAuthErr(err, "grant", "get", models.InfraAdminRole)
	}

	return data.GetGrant(db, data.GetGrantOptions{ByID: id})
}

func ListGrants(c *gin.Context, opts data.ListGrantsOptions, lastUpdateIndex int64) (data.ListGrantsResponse, error) {
	rCtx := GetRequestContext(c)
	resp := data.ListGrantsResponse{}
	subject := opts.BySubject

	roles := []string{models.InfraAdminRole, models.InfraViewRole, models.InfraConnectorRole}
	_, err := RequireInfraRole(c, roles...)
	err = HandleAuthErr(err, "grants", "list", roles...)
	if errors.Is(err, ErrNotAuthorized) {
		// Allow an authenticated identity to view their own grants
		subjectID, _ := subject.ID() // zero value will never match a user
		switch {
		case rCtx.Authenticated.User == nil:
			return resp, err
		case subject.IsIdentity() && rCtx.Authenticated.User.ID == subjectID:
			// authorized because the request is for their own grants
		case subject.IsGroup() && userInGroup(rCtx.DBTxn, rCtx.Authenticated.User.ID, subjectID):
			// authorized because the request is for grants of a group they belong to
		default:
			return resp, err
		}
	} else if err != nil {
		return resp, err
	}

	if lastUpdateIndex == 0 {
		return data.ListGrants(rCtx.DBTxn, opts)
	}

	// TODO: validate that only supported query parameters are set, and that at least
	// one of the required parameters are set with lastUpdateIndex
	// TODO: change request timeout for these requests
	listenOpts := data.ListenGrantsOptions{ByResource: opts.ByResource}
	listener, err := data.ListenForGrantsNotify(rCtx.Request.Context(), rCtx.DataDB, listenOpts)
	if err != nil {
		return resp, fmt.Errorf("listen for notify: %w", err)
	}
	defer func() {
		// use a context with a separate deadline so that we still release
		// when the request timeout is reached
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()
		if err := listener.Release(ctx); err != nil {
			logging.L.Error().Err(err).Msg("failed to release listener conn")
		}
	}()

	response, err := data.ListGrants(rCtx.DBTxn, opts)
	if err != nil {
		return resp, err
	}

	// The query returned results that are new to the client
	if response.MaxUpdateIndex > lastUpdateIndex {
		return response, nil
	}

	_, err = listener.WaitForNotification(rCtx.Request.Context())
	if err != nil {
		return resp, fmt.Errorf("waiting for notify: %w", err)
	}

	tx, err := rCtx.DataDB.Begin(rCtx.Request.Context(), &sql.TxOptions{
		Isolation: sql.LevelRepeatableRead,
		ReadOnly:  true,
	})
	if err != nil {
		return resp, err
	}
	defer func() {
		if err := tx.Rollback(); err != nil {
			logging.L.Warn().Err(err).Msg("rollback failed")
		}
	}()

	response, err = data.ListGrants(tx, opts)
	if err != nil {
		return resp, err
	}

	// TODO: check if the maxIndex > lastUpdateIndex, and start waiting for
	// notification again when it's false. When we include group membership
	// changes in the query this will be an optimization.

	return response, nil
}

func userInGroup(db data.GormTxn, authnUserID uid.ID, groupID uid.ID) bool {
	groups, err := data.ListGroups(db, &data.Pagination{Limit: 1}, data.ByGroupMember(authnUserID), data.ByID(groupID))
	if err != nil {
		return false
	}

	for _, g := range groups {
		if g.ID == groupID {
			return true
		}
	}
	return false
}

func CreateGrant(c *gin.Context, grant *models.Grant) error {
	rCtx := GetRequestContext(c)

	var err error
	if grant.Privilege == models.InfraSupportAdminRole && grant.Resource == ResourceInfraAPI {
		_, err = RequireInfraRole(c, models.InfraSupportAdminRole)
	} else {
		_, err = RequireInfraRole(c, models.InfraAdminRole)
	}

	if err != nil {
		return HandleAuthErr(err, "grant", "create", grant.Privilege)
	}

	// TODO: CreatedBy should be set automatically
	grant.CreatedBy = rCtx.Authenticated.User.ID

	return data.CreateGrant(rCtx.DBTxn, grant)
}

func DeleteGrant(c *gin.Context, id uid.ID) error {
	db, err := RequireInfraRole(c, models.InfraAdminRole)
	if err != nil {
		return HandleAuthErr(err, "grant", "delete", models.InfraAdminRole)
	}

	return data.DeleteGrants(db, data.DeleteGrantsOptions{ByID: id})
}
