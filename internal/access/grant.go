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

type ListGrantsResponse struct {
	Grants         []models.Grant
	MaxUpdateIndex int64
}

func ListGrants(c *gin.Context, opts data.ListGrantsOptions, lastUpdateIndex int64) (ListGrantsResponse, error) {
	rCtx := GetRequestContext(c)
	subject := opts.BySubject

	roles := []string{models.InfraAdminRole, models.InfraViewRole, models.InfraConnectorRole}
	_, err := RequireInfraRole(c, roles...)
	err = HandleAuthErr(err, "grants", "list", roles...)
	if errors.Is(err, ErrNotAuthorized) {
		// Allow an authenticated identity to view their own grants
		subjectID, _ := subject.ID() // zero value will never match a user
		switch {
		case rCtx.Authenticated.User == nil:
			return ListGrantsResponse{}, err
		case subject.IsIdentity() && rCtx.Authenticated.User.ID == subjectID:
			// authorized because the request is for their own grants
		case subject.IsGroup() && userInGroup(rCtx.DBTxn, rCtx.Authenticated.User.ID, subjectID):
			// authorized because the request is for grants of a group they belong to
		default:
			return ListGrantsResponse{}, err
		}
	} else if err != nil {
		return ListGrantsResponse{}, err
	}

	if lastUpdateIndex == 0 {
		result, err := data.ListGrants(rCtx.DBTxn, opts)
		return ListGrantsResponse{Grants: result}, err
	}

	// Close the request scoped txn to avoid long-running transactions.
	if err := rCtx.DBTxn.Rollback(); err != nil {
		return ListGrantsResponse{}, err
	}

	listenOpts := data.ListenForGrantsOptions{
		ByDestination: opts.ByDestination,
		OrgID:         rCtx.DBTxn.OrganizationID(),
	}
	listener, err := data.ListenForGrantsNotify(rCtx.Request.Context(), rCtx.DataDB, listenOpts)
	if err != nil {
		return ListGrantsResponse{}, fmt.Errorf("listen for notify: %w", err)
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

	result, err := listGrantsWithMaxUpdateIndex(rCtx, opts)
	if err != nil {
		return result, err
	}

	// The query returned results that are new to the client
	if result.MaxUpdateIndex > lastUpdateIndex {
		return result, nil
	}

	err = listener.WaitForNotification(rCtx.Request.Context())
	if err != nil {
		return result, fmt.Errorf("waiting for notify: %w", err)
	}

	result, err = listGrantsWithMaxUpdateIndex(rCtx, opts)
	if err != nil {
		return result, err
	}

	// TODO: check if the maxIndex > lastUpdateIndex, and start waiting for
	// notification again when it's false. When we include group membership
	// changes in the query this will be an optimization.
	return result, nil
}

func listGrantsWithMaxUpdateIndex(rCtx RequestContext, opts data.ListGrantsOptions) (ListGrantsResponse, error) {
	tx, err := rCtx.DataDB.Begin(rCtx.Request.Context(), &sql.TxOptions{
		ReadOnly:  true,
		Isolation: sql.LevelRepeatableRead,
	})
	if err != nil {
		return ListGrantsResponse{}, err
	}
	defer logError(tx.Rollback, "failed to rollback transaction")
	tx = tx.WithOrgID(rCtx.DBTxn.OrganizationID())

	result, err := data.ListGrants(tx, opts)
	if err != nil {
		return ListGrantsResponse{}, err
	}

	maxUpdateIndex, err := data.GrantsMaxUpdateIndex(tx, data.GrantsMaxUpdateIndexOptions{
		ByDestination: opts.ByDestination,
	})
	return ListGrantsResponse{Grants: result, MaxUpdateIndex: maxUpdateIndex}, err
}

func logError(fn func() error, msg string) {
	if err := fn(); err != nil {
		logging.L.Warn().Err(err).Msg(msg)
	}
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

func UpdateGrants(c *gin.Context, grants []*models.Grant) error {
	db, err := RequireInfraRole(c, models.InfraAdminRole)
	if err != nil {
		return HandleAuthErr(err, "grant", "create", "")
	}

	err = data.CreateGrants(db, grants)
	if err != nil {
		return err
	}

	return nil
}
