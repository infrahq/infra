package server

import (
	"net/http"
	"net/http/pprof"

	"github.com/gin-gonic/gin"

	"github.com/infrahq/infra/internal/access"
	"github.com/infrahq/infra/internal/server/models"
)

func pprofHandler(c *gin.Context) {
	if _, err := access.RequireInfraRole(c, models.InfraSupportAdminRole); err != nil {
		sendAPIError(c, access.HandleAuthErr(err, "debug", "run", models.InfraSupportAdminRole))
		return
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
}
