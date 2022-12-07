package access

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/logging"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

func CreateCredentialRequest(c *gin.Context, destination string) (*models.CredentialRequest, error) {
	rCtx := GetRequestContext(c)
	// does the user have a grant to this destination?
	// ListGrants(c, )
	grants, err := data.ListGrants(rCtx.DBTxn, data.ListGrantsOptions{
		ByDestination: destination,
		BySubject:     uid.NewIdentityPolymorphicID(rCtx.Authenticated.User.ID),
		Pagination: &data.Pagination{
			Limit: 1,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("ListGrants: %w", err)
	}
	if len(grants) == 0 {
		return nil, internal.ErrUnauthorized
	}

	dest, err := data.GetDestination(rCtx.DBTxn, data.GetDestinationOptions{ByName: destination})
	if err != nil {
		return nil, fmt.Errorf("GetDestination: %w", err)
	}

	cr := &models.CredentialRequest{
		ID:                 uid.New(),
		OrganizationMember: models.OrganizationMember{OrganizationID: rCtx.Authenticated.User.OrganizationID},
		ExpiresAt:          time.Now().Add(2 * time.Minute),
		UserID:             rCtx.Authenticated.User.ID,
		DestinationID:      dest.ID,
	}
	err = data.CreateCredentialRequest(rCtx.DBTxn, cr)
	if err != nil {
		return nil, fmt.Errorf("CreateCredentialRequest: %w", err)
	}
	logging.Debugf("Creating CredentialRequest with id %d, orgID %d", cr.ID, cr.OrganizationID)

	// commit the transaction so others can see the request
	err = rCtx.DBTxn.Commit()
	if err != nil {
		return nil, err
	}

	// wait for the request to be filled; listen for the notify on the specific record req.ID
	listener, err := data.ListenForNotify(c, rCtx.DataDB, data.ListenForNotifyOptions{
		OrgID:                  rCtx.Authenticated.AccessKey.OrganizationID,
		CredentialRequestsByID: cr.ID,
	})
	if err != nil {
		return nil, fmt.Errorf("ListenForNotify: %w", err)
	}
	defer func() {
		err := listener.Release(c)
		if err != nil {
			logging.Errorf("error releasing listener: %s", err)
		}
	}()

	err = listener.WaitForNotification(c)
	if err != nil {
		logging.Debugf("CreateCredentialRequest notification timeout")
		return nil, fmt.Errorf("WaitForNotification %w: %s", api.ErrTimeout, err)
	}
	logging.Debugf("CreateCredentialRequest notification received")

	tx, err := rCtx.DataDB.Begin(rCtx.Request.Context(), &sql.TxOptions{
		Isolation: sql.LevelReadCommitted,
		ReadOnly:  true,
	})
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	tx = tx.WithOrgID(cr.OrganizationID)

	// looks like the CredentialRequest has been updated; reload the credential request
	logging.Debugf("Reloading CredentialRequest with id %d, orgID %d", cr.ID, cr.OrganizationID)
	cr, err = data.GetCredentialRequest(tx, cr.ID, cr.OrganizationID)
	if err != nil {
		return nil, fmt.Errorf("GetCredentialRequest: %w", err)
	}
	if err = tx.Rollback(); err != nil {
		logging.Debugf("could not roll back transaction")
	}

	return cr, nil
}

func ListCredentialRequests(c *gin.Context, destination string, lastUpdateIndex int64) (ListCredentialRequestResponse, error) {
	rCtx := GetRequestContext(c)

	roles := []string{models.InfraAdminRole, models.InfraViewRole, models.InfraConnectorRole}
	_, err := RequireInfraRole(c, roles...)
	err = HandleAuthErr(err, "grants", "list", roles...)
	if err != nil {
		return ListCredentialRequestResponse{}, err
	}

	dest, err := data.GetDestination(rCtx.DBTxn, data.GetDestinationOptions{ByName: destination})
	if err != nil {
		return ListCredentialRequestResponse{}, err
	}

	if lastUpdateIndex == 0 {
		logging.Debugf("listing all credential requests")
		result, err := data.ListCredentialRequests(rCtx.DBTxn, dest.ID)
		if len(result) > 0 {
			return ListCredentialRequestResponse{Items: result}, err
		}
	}

	// Close the request scoped txn to avoid long-running transactions.
	if err := rCtx.DBTxn.Rollback(); err != nil {
		return ListCredentialRequestResponse{}, err
	}

	listenOpts := data.ListenForNotifyOptions{
		CredentialRequestsByDestinationID: dest.ID,
		OrgID:                             rCtx.DBTxn.OrganizationID(),
	}
	query := func() (ListCredentialRequestResponse, error) {
		return listCredentialRequestsWithMaxUpdateIndex(rCtx, dest.ID)
	}
	return blockingRequest(rCtx, listenOpts, query, lastUpdateIndex)
}

type ListCredentialRequestResponse struct {
	Items          []models.CredentialRequest
	MaxUpdateIndex int64
}

func (l ListCredentialRequestResponse) UpdateIndex() int64 {
	return l.MaxUpdateIndex
}

func listCredentialRequestsWithMaxUpdateIndex(rCtx RequestContext, destinationID uid.ID) (ListCredentialRequestResponse, error) {
	tx, err := rCtx.DataDB.Begin(rCtx.Request.Context(), &sql.TxOptions{
		ReadOnly:  true,
		Isolation: sql.LevelRepeatableRead,
	})
	if err != nil {
		return ListCredentialRequestResponse{}, err
	}
	defer logError(tx.Rollback, "failed to rollback transaction")
	tx = tx.WithOrgID(rCtx.DBTxn.OrganizationID())

	result, err := data.ListCredentialRequests(tx, destinationID)
	if err != nil {
		return ListCredentialRequestResponse{}, err
	}

	maxUpdateIndex, err := data.CredentialRequestsMaxUpdateIndex(tx, destinationID)
	return ListCredentialRequestResponse{Items: result, MaxUpdateIndex: maxUpdateIndex}, err
}

func UpdateCredentialRequest(rctx RequestContext, r *api.UpdateCredentialRequest) error {
	cr, err := data.GetCredentialRequest(rctx.DBTxn, r.ID, r.OrganizationID)
	if err != nil {
		return err
	}

	cr.FromUpdateAPI(r)

	err = data.UpdateCredentialRequest(rctx.DBTxn, cr)
	if err != nil {
		return err
	}

	return nil
}
