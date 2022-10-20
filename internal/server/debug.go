package server

import (
	"database/sql"
	"net/http"
	"net/http/pprof"

	"github.com/gin-gonic/gin"

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

func pprofHandler(c *gin.Context, _ *pprofRequest) (*api.EmptyResponse, error) {
	rCtx := getRequestContext(c)
	if _, err := access.RequireInfraRole(c, models.InfraSupportAdminRole); err != nil {
		return nil, access.HandleAuthErr(err, "debug", "run", models.InfraSupportAdminRole)
	}
	// end the transaction before blocking
	if err := rCtx.DBTxn.Rollback(); err != nil {
		return nil, err
	}

	switch c.Param("profile") {
	case "/trace":
		pprof.Trace(c.Writer, c.Request)
	case "/profile":
		pprof.Profile(c.Writer, c.Request)
	default:
		// All other types of profiles are served from Index
		http.StripPrefix("/api", http.HandlerFunc(pprof.Index)).ServeHTTP(c.Writer, c.Request)
	}
	return nil, nil
}
