package access

import (
	"database/sql"
	"errors"

	"github.com/infrahq/infra/internal/logging"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

func GetGrant(rCtx RequestContext, id uid.ID) (*models.Grant, error) {
	err := IsAuthorized(rCtx, models.InfraAdminRole)
	if err != nil {
		return nil, HandleAuthErr(err, "grant", "get", models.InfraAdminRole)
	}

	return data.GetGrant(rCtx.DBTxn, data.GetGrantOptions{ByID: id})
}

type ListGrantsResponse struct {
	Grants         []models.Grant
	MaxUpdateIndex int64
}

func ListGrants(rCtx RequestContext, opts data.ListGrantsOptions, lastUpdateIndex int64) (ListGrantsResponse, error) {
	subject := opts.BySubject

	roles := []string{models.InfraAdminRole, models.InfraViewRole, models.InfraConnectorRole}
	err := IsAuthorized(rCtx, roles...)
	err = HandleAuthErr(err, "grants", "list", roles...)
	if errors.Is(err, ErrNotAuthorized) {
		// Allow an authenticated identity to view their own grants
		switch {
		case rCtx.Authenticated.User == nil:
			return ListGrantsResponse{}, err
		case subject.Kind == models.SubjectKindUser && rCtx.Authenticated.User.ID == subject.ID:
			// authorized because the request is for their own grants
		case subject.Kind == models.SubjectKindGroup && userInGroup(rCtx.DBTxn, rCtx.Authenticated.User.ID, subject.ID):
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

	dest, err := data.GetDestination(rCtx.DBTxn, data.GetDestinationOptions{ByName: opts.ByDestination})
	if err != nil {
		return ListGrantsResponse{}, err
	}

	// Close the request scoped txn to avoid long-running transactions.
	if err := rCtx.DBTxn.Rollback(); err != nil {
		return ListGrantsResponse{}, err
	}

	listenOpts := data.ListenChannelGrantsByDestination{
		DestinationID: dest.ID,
		OrgID:         rCtx.DBTxn.OrganizationID(),
	}
	query := &listGrantsBlockingQuery{
		do: func() (ListGrantsResponse, error) {
			return listGrantsWithMaxUpdateIndex(rCtx, opts)
		},
		previousUpdateIndex: lastUpdateIndex,
	}
	err = RunBlockingRequest(rCtx, query, listenOpts)
	return query.result, err
}

type listGrantsBlockingQuery struct {
	previousUpdateIndex int64
	do                  func() (ListGrantsResponse, error)
	result              ListGrantsResponse
}

func (q *listGrantsBlockingQuery) Do() error {
	var err error
	q.result, err = q.do()
	return err
}

func (q *listGrantsBlockingQuery) IsDone() bool {
	return q.result.MaxUpdateIndex > q.previousUpdateIndex
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

func userInGroup(db data.ReadTxn, authnUserID uid.ID, groupID uid.ID) bool {
	groups, err := data.ListGroupIDsForUser(db, authnUserID)
	if err != nil {
		return false
	}

	for _, g := range groups {
		if g == groupID {
			return true
		}
	}
	return false
}

func CreateGrant(rCtx RequestContext, grant *models.Grant) error {
	role := requiredInfraRoleForGrantOperation(grant)
	err := IsAuthorized(rCtx, role)
	if err != nil {
		return HandleAuthErr(err, "grant", "create", role)
	}

	// TODO: CreatedBy should be set automatically
	grant.CreatedBy = rCtx.Authenticated.User.ID

	return data.CreateGrant(rCtx.DBTxn, grant)
}

func DeleteGrant(rCtx RequestContext, id uid.ID) error {
	// TODO: should support-admin role be required to delete support-admin grant?
	err := IsAuthorized(rCtx, models.InfraAdminRole)
	if err != nil {
		return HandleAuthErr(err, "grant", "delete", models.InfraAdminRole)
	}

	return data.DeleteGrants(rCtx.DBTxn, data.DeleteGrantsOptions{ByID: id})
}

func UpdateGrants(rCtx RequestContext, addGrants, rmGrants []*models.Grant) error {
	all := make([]*models.Grant, 0, len(addGrants)+len(rmGrants))
	all = append(all, addGrants...)
	all = append(all, rmGrants...)
	role := requiredInfraRoleForGrantOperation(all...)
	err := IsAuthorized(rCtx, role)
	if err != nil {
		return HandleAuthErr(err, "grant", "update", role)
	}

	return data.UpdateGrants(rCtx.DBTxn, addGrants, rmGrants)
}

func requiredInfraRoleForGrantOperation(grants ...*models.Grant) string {
	for _, grant := range grants {
		if grant.Privilege == models.InfraSupportAdminRole && grant.Resource == ResourceInfraAPI {
			return models.InfraSupportAdminRole
		}
	}
	return models.InfraAdminRole
}
