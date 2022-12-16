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

type ListDestinationCredentialResponse struct {
	Items          []models.DestinationCredential
	MaxUpdateIndex int64
}

func (l ListDestinationCredentialResponse) UpdateIndex() int64 {
	return l.MaxUpdateIndex
}

func (l ListDestinationCredentialResponse) ItemCount() int {
	return len(l.Items)
}

func CreateDestinationCredential(c *gin.Context, destination string) (*models.DestinationCredential, error) {
	rCtx := GetRequestContext(c)
	// does the user have a grant to this destination?
	// ListGrants(c, )
	grants, err := data.ListGrants(rCtx.DBTxn, data.ListGrantsOptions{
		ByDestination:              destination,
		BySubject:                  uid.NewIdentityPolymorphicID(rCtx.Authenticated.User.ID),
		IncludeInheritedFromGroups: true,
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

	cr := &models.DestinationCredential{
		ID:                 uid.New(),
		OrganizationMember: models.OrganizationMember{OrganizationID: rCtx.Authenticated.User.OrganizationID},
		RequestExpiresAt:   time.Now().Add(2 * time.Minute),
		UserID:             rCtx.Authenticated.User.ID,
		DestinationID:      dest.ID,
	}
	err = data.CreateDestinationCredential(rCtx.DBTxn, cr)
	if err != nil {
		return nil, fmt.Errorf("CreateDestinationCredential: %w", err)
	}
	logging.Debugf("Creating DestinationCredential with id %d, orgID %d", cr.ID, cr.OrganizationID)

	// commit the transaction so others can see the request
	err = rCtx.DBTxn.Commit()
	if err != nil {
		return nil, err
	}

	// wait for the request to be filled; listen for the notify on the specific record req.ID
	listener, err := data.ListenForNotify(c, rCtx.DataDB, data.ListenForNotifyOptions{
		OrgID:                      rCtx.Authenticated.AccessKey.OrganizationID,
		DestinationCredentialsByID: cr.ID,
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
		logging.Debugf("CreateDestinationCredential notification timeout")
		return nil, fmt.Errorf("WaitForNotification %w: %s", api.ErrTimeout, err)
	}
	logging.Debugf("CreateDestinationCredential notification received")

	tx, err := rCtx.DataDB.Begin(rCtx.Request.Context(), &sql.TxOptions{
		Isolation: sql.LevelReadCommitted,
		ReadOnly:  true,
	})
	if err != nil {
		return nil, fmt.Errorf("n: %w", err)
	}
	tx = tx.WithOrgID(cr.OrganizationID)

	// looks like the DestinationCredential has been updated; reload the destination credential
	logging.Debugf("Reloading DestinationCredential with id %d, orgID %d", cr.ID, cr.OrganizationID)
	cr, err = data.GetDestinationCredential(tx, cr.ID, cr.OrganizationID)
	if err != nil {
		return nil, fmt.Errorf("GetDestinationCredential: %w", err)
	}
	if err = tx.Rollback(); err != nil {
		logging.Debugf("could not roll back transaction")
	}

	return cr, nil
}

func ListDestinationCredentials(c *gin.Context, destination string, lastUpdateIndex int64) (ListDestinationCredentialResponse, error) {
	rCtx := GetRequestContext(c)

	roles := []string{models.InfraAdminRole, models.InfraViewRole, models.InfraConnectorRole}
	_, err := RequireInfraRole(c, roles...)
	err = HandleAuthErr(err, "grants", "list", roles...)
	if err != nil {
		return ListDestinationCredentialResponse{}, err
	}

	dest, err := data.GetDestination(rCtx.DBTxn, data.GetDestinationOptions{ByName: destination})
	if err != nil {
		return ListDestinationCredentialResponse{}, err
	}

	if lastUpdateIndex == 0 {
		logging.Debugf("listing all destination credentials")
		result, err := data.ListDestinationCredentials(rCtx.DBTxn, dest.ID)
		if err != nil {
			return ListDestinationCredentialResponse{}, err
		}
		if len(result) > 0 {
			return ListDestinationCredentialResponse{Items: result}, err
		}
	}

	// Close the request scoped txn to avoid long-running transactions.
	if err := rCtx.DBTxn.Rollback(); err != nil {
		return ListDestinationCredentialResponse{}, err
	}

	listenOpts := data.ListenForNotifyOptions{
		DestinationCredentialsByDestinationID: dest.ID,
		OrgID:                                 rCtx.DBTxn.OrganizationID(),
	}
	query := func() (ListDestinationCredentialResponse, error) {
		return listDestinationCredentialsWithMaxUpdateIndex(rCtx, dest.ID)
	}
	return blockingRequest(rCtx, listenOpts, query, lastUpdateIndex)
}

type ListDestinationCredentialRespct struct {
	Items          []models.DestinationCredential
	MaxUpdateIndex int64
}

func listDestinationCredentialsWithMaxUpdateIndex(rCtx RequestContext, destinationID uid.ID) (ListDestinationCredentialResponse, error) {
	tx, err := rCtx.DataDB.Begin(rCtx.Request.Context(), &sql.TxOptions{
		ReadOnly:  true,
		Isolation: sql.LevelRepeatableRead,
	})
	if err != nil {
		return ListDestinationCredentialResponse{}, err
	}
	defer logError(tx.Rollback, "failed to rollback transaction")
	tx = tx.WithOrgID(rCtx.DBTxn.OrganizationID())

	result, err := data.ListDestinationCredentials(tx, destinationID)
	if err != nil {
		return ListDestinationCredentialResponse{}, err
	}

	maxUpdateIndex, err := data.DestinationCredentialsMaxUpdateIndex(tx, destinationID)
	return ListDestinationCredentialResponse{Items: result, MaxUpdateIndex: maxUpdateIndex}, err
}

func AnswerDestinationCredential(rctx RequestContext, r *api.AnswerDestinationCredential) error {
	cr, err := data.GetDestinationCredential(rctx.DBTxn, r.ID, r.OrganizationID)
	if err != nil {
		return err
	}

	cr.FromUpdateAPI(r)

	err = data.AnswerDestinationCredential(rctx.DBTxn, cr)
	if err != nil {
		return err
	}

	return nil
}
