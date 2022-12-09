package server

import (
	"encoding/base64"
	"fmt"

	"github.com/gin-gonic/gin"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal/logging"
	"github.com/infrahq/infra/internal/server/data"
)

var verifyAndRedirectRoute = route[api.VerifyAndRedirectRequest, *api.RedirectResponse]{
	handler: VerifyAndRedirect,
	routeSettings: routeSettings{
		omitFromDocs:               true,
		omitFromTelemetry:          true,
		infraVersionHeaderOptional: true,
	},
}

func VerifyAndRedirect(c *gin.Context, r *api.VerifyAndRedirectRequest) (*api.RedirectResponse, error) {
	// No authorization required
	rCtx := getRequestContext(c)
	if err := data.SetIdentityVerified(rCtx.DBTxn, r.VerificationToken); err != nil {
		logging.L.Error().Msg("VerifyUserByToken: " + err.Error())
	}

	redirectTo, err := base64.URLEncoding.DecodeString(r.Base64RedirectURL)
	if err != nil {
		return nil, fmt.Errorf("decoding redirect url: %w", err)
	}

	return &api.RedirectResponse{
		RedirectTo: string(redirectTo),
	}, nil
}
