package server

import (
	"database/sql"
	"net/http"
	"net/http/pprof"
	"path"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal/access"
	"github.com/infrahq/infra/internal/server/models"
)

var pprofRoute = route[pprofRequest, *api.EmptyResponse]{
	handler: pprofHandler,
	routeSettings: routeSettings{
		omitFromTelemetry:          true,
		omitFromDocs:               true,
		infraVersionHeaderOptional: true,
		txnOptions:                 &sql.TxOptions{ReadOnly: true},
	},
}

type pprofRequest struct{}

func (pprofRequest) IsBlockingRequest() bool {
	return true
}

func pprofHandler(rCtx access.RequestContext, _ *pprofRequest) (*api.EmptyResponse, error) {
	if err := access.IsAuthorized(rCtx, models.InfraSupportAdminRole); err != nil {
		return nil, access.HandleAuthErr(err, "debug", "run", models.InfraSupportAdminRole)
	}
	// end the transaction before blocking
	if err := rCtx.DBTxn.Rollback(); err != nil {
		return nil, err
	}

	_, profile := path.Split(rCtx.Request.URL.Path)
	switch profile {
	case "trace":
		pprof.Trace(rCtx.Response.HTTPWriter, rCtx.Request)
	case "profile":
		pprof.Profile(rCtx.Response.HTTPWriter, rCtx.Request)
	default:
		// All other types of profiles are served from Index
		http.StripPrefix("/api", http.HandlerFunc(pprof.Index)).ServeHTTP(rCtx.Response.HTTPWriter, rCtx.Request)
	}
	return nil, nil
}
