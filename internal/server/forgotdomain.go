package server

import (
	"errors"

	"github.com/gin-gonic/gin"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/access"
	"github.com/infrahq/infra/internal/logging"
	//"github.com/infrahq/infra/internal/server/email"
)

func (a *API) RequestForgotDomains(c *gin.Context, r *api.ForgotDomainRequest) (*api.EmptyResponse, error) {
	logging.Infof("forgot domain")
	domains, err := access.ForgotDomainRequest(c, r.Email)

	if err != nil {
		if errors.Is(err, internal.ErrNotFound) {
			logging.Infof("No orgs found for user %s", r.Email)
			return nil, nil // This is okay. we don't notify the user if we failed to find the email.
		}
		return nil, err
	}

	// TODO: Send the email
	for _, d := range domains {
		logging.Infof("Forgot domain: %s -- '%s' '%s'", r.Email, d.OrganizationName, d.OrganizationDomain)
	}

	return nil, nil
}
