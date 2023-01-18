package server

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal/access"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

func (a *API) ListDestinations(c *gin.Context, r *api.ListDestinationsRequest) (*api.ListResponse[api.Destination], error) {
	rCtx := getRequestContext(c)
	p := PaginationFromRequest(r.PaginationRequest)

	opts := data.ListDestinationsOptions{
		ByUniqueID: r.UniqueID,
		ByName:     r.Name,
		ByKind:     r.Kind,
		Pagination: &p,
	}
	destinations, err := data.ListDestinations(rCtx.DBTxn, opts)
	if err != nil {
		return nil, err
	}

	result := api.NewListResponse(destinations, PaginationToResponse(p), func(destination models.Destination) api.Destination {
		return *destination.ToAPI()
	})

	return result, nil
}

func (a *API) GetDestination(c *gin.Context, r *api.Resource) (*api.Destination, error) {
	// No authorization required to view a destination
	rCtx := getRequestContext(c)
	destination, err := data.GetDestination(rCtx.DBTxn, data.GetDestinationOptions{ByID: r.ID})
	if err != nil {
		return nil, err
	}

	return destination.ToAPI(), nil
}

func (a *API) CreateDestination(c *gin.Context, r *api.CreateDestinationRequest) (*api.Destination, error) {
	rCtx := getRequestContext(c)
	destination := &models.Destination{
		Name:          r.Name,
		UniqueID:      r.UniqueID,
		Kind:          models.DestinationKind(r.Kind),
		ConnectionURL: r.Connection.URL,
		ConnectionCA:  string(r.Connection.CA),
		Resources:     r.Resources,
		Roles:         r.Roles,
		Version:       r.Version,
	}

	if destination.Kind == "" {
		destination.Kind = "kubernetes"
	}

	// set LastSeenAt if this request came from a connector. The middleware
	// can't do this update in the case where the destination did not exist yet
	switch {
	case rCtx.Request.Header.Get(headerInfraDestinationName) == r.Name:
		destination.LastSeenAt = time.Now()
	case rCtx.Request.Header.Get(headerInfraDestinationUniqueID) == r.UniqueID:
		destination.LastSeenAt = time.Now()
	}

	err := access.CreateDestination(rCtx, destination)
	if err != nil {
		return nil, fmt.Errorf("create destination: %w", err)
	}

	return destination.ToAPI(), nil
}

func (a *API) UpdateDestination(c *gin.Context, r *api.UpdateDestinationRequest) (*api.Destination, error) {
	rCtx := getRequestContext(c)

	// Start with the existing value, so that non-update fields are not set to zero.
	destination, err := data.GetDestination(rCtx.DBTxn, data.GetDestinationOptions{ByID: r.ID})
	if err != nil {
		return nil, err
	}

	destination.Name = r.Name
	destination.UniqueID = r.UniqueID
	destination.ConnectionURL = r.Connection.URL
	destination.ConnectionCA = string(r.Connection.CA)
	destination.Resources = r.Resources
	destination.Roles = r.Roles
	destination.Version = r.Version

	if err := access.UpdateDestination(rCtx, destination); err != nil {
		return nil, fmt.Errorf("update destination: %w", err)
	}

	return destination.ToAPI(), nil
}

func (a *API) DeleteDestination(c *gin.Context, r *api.Resource) (*api.EmptyResponse, error) {
	return nil, access.DeleteDestination(getRequestContext(c), r.ID)
}

// TODO: move types to api package
type ListDestinationAccessRequest struct {
	Name string `uri:"name"` // TODO: change to ID when grants stores destinationID
	api.BlockingRequest
}

type ListDestinationAccessResponse struct {
	Items               []DestinationAccess
	api.LastUpdateIndex `json:"-"`
}

type DestinationAccess struct {
	UserID           uid.ID
	UserSSHLoginName string
	Privilege        string
	Resource         string
}

func ListDestinationAccess(c *gin.Context, r *ListDestinationAccessRequest) (*ListDestinationAccessResponse, error) {
	rCtx := getRequestContext(c)
	rCtx.Response.AddLogFields(func(event *zerolog.Event) {
		event.Int64("lastUpdateIndex", r.LastUpdateIndex)
	})

	roles := []string{models.InfraAdminRole, models.InfraViewRole, models.InfraConnectorRole}
	if err := access.IsAuthorized(rCtx, roles...); err != nil {
		return nil, access.HandleAuthErr(err, "grants", "list", roles...)
	}

	if r.LastUpdateIndex == 0 {
		result, err := data.ListDestinationAccess(rCtx.DBTxn, r.Name)
		if err != nil {
			return nil, err
		}
		return &api.ListDestinationAccessResponse{
			Items: destinationAccessToAPI(result.Items),
		}, nil
	}

	dest, err := data.GetDestination(rCtx.DBTxn, data.GetDestinationOptions{ByName: r.Name})
	if err != nil {
		return nil, err
	}

	grants, err := data.ListGrants(rCtx.DBTxn, data.ListGrantsOptions{ByDestination: r.Name})
	if err != nil {
		return nil, err
	}

	// Close the request scoped txn to avoid long-running transactions.
	if err := rCtx.DBTxn.Rollback(); err != nil {
		return nil, err
	}

	channels := []data.ListenChannelDescriptor{
		data.ListenChannelGrantsByDestination{
			OrgID:         rCtx.DBTxn.OrganizationID(),
			DestinationID: dest.ID,
		},
	}
	for _, grant := range grants {
		if grant.Subject.Kind == models.SubjectKindGroup {
			channels = append(channels, data.ListenChannelGroupMembership{
				OrgID:   rCtx.DBTxn.OrganizationID(),
				GroupID: grant.Subject.ID,
			})
		}
	}
	query := &destinationAccessQuery{
		previousUpdateIndex: r.LastUpdateIndex,
		do: func() (*data.ListDestinationAccessResult, error) {
			return listDestinationAccessWithMaxUpdateIndex(rCtx, r.Name)
		},
	}
	if err = access.RunBlockingRequest(rCtx, query, channels...); err != nil {
		return nil, err
	}

	rCtx.Response.AddLogFields(func(event *zerolog.Event) {
		event.Int("numItems", len(query.result.Items))
	})

	return &api.ListDestinationAccessResponse{
		Items:           destinationAccessToAPI(query.result.Items),
		LastUpdateIndex: api.LastUpdateIndex{Index: query.result.MaxUpdateIndex},
	}, nil

}

type destinationAccessQuery struct {
	previousUpdateIndex int64
	do                  func() (*data.ListDestinationAccessResult, error)
	result              *data.ListDestinationAccessResult
}

func (q *destinationAccessQuery) Do() error {
	var err error
	q.result, err = q.do()
	return err
}

func (q *destinationAccessQuery) IsDone() bool {
	if q.result == nil {
		return false
	}
	return q.result.MaxUpdateIndex > q.previousUpdateIndex
}

func destinationAccessToAPI(a []data.DestinationAccess) []DestinationAccess {
	result := make([]DestinationAccess, 0, len(a))
	for _, item := range a {
		result = append(result, DestinationAccess{
			UserID:           item.UserID,
			UserSSHLoginName: item.UserSSHLoginName,
			Privilege:        item.Privilege,
			Resource:         item.Resource,
		})
	}
	return result
}

func listDestinationAccessWithMaxUpdateIndex(rCtx access.RequestContext, name string) (*data.ListDestinationAccessResult, error) {
	tx, err := rCtx.DataDB.Begin(rCtx.Request.Context(), &sql.TxOptions{
		ReadOnly:  true,
		Isolation: sql.LevelRepeatableRead,
	})
	if err != nil {
		return nil, err
	}
	defer logError(tx.Rollback, "failed to rollback transaction")
	tx = tx.WithOrgID(rCtx.DBTxn.OrganizationID())

	return data.ListDestinationAccess(tx, name)
}
