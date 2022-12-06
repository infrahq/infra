package server

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/access"
	"github.com/infrahq/infra/internal/logging"
	"github.com/infrahq/infra/internal/server/authn"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/email"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/internal/server/providers"
)

func (a *API) SignupRoute() route[api.SignupRequest, *api.SignupResponse] {
	return route[api.SignupRequest, *api.SignupResponse]{
		handler: a.Signup,
		routeSettings: routeSettings{
			omitFromDocs: true,
		},
	}
}

func (a *API) Signup(c *gin.Context, r *api.SignupRequest) (*api.SignupResponse, error) {
	if !a.server.options.EnableSignup {
		return nil, fmt.Errorf("%w: signup is disabled", internal.ErrBadRequest)
	}

	keyExpires := time.Now().UTC().Add(a.server.options.SessionDuration)

	var created *access.NewOrgDetails
	switch {
	case r.Social != nil:
		// do social sign-up
		if a.server.Google == nil {
			return nil, fmt.Errorf("%w: google login is not configured, provider id must be specified for oidc login", internal.ErrBadRequest)
		}
		// check if an org exists with their desired sub-domain
		// this has to be done here since the auth code is single-use
		if !access.DomainAvailable(c, fmt.Sprintf("%s.%s", r.Subdomain, a.server.options.BaseDomain)) {
			return nil, fmt.Errorf("%w: domain is not available", internal.ErrBadRequest)
		}
		// perform OIDC authentication
		provider := a.server.Google
		auth := &authn.OIDCAuthn{
			RedirectURL: r.Social.RedirectURL,
			Code:        r.Social.Code,
		}
		idpAuth, err := a.socialSignupUserAuth(c, provider, auth)
		if err != nil {
			return nil, err // make sure to return this error directly for an unauthorized response
		}
		details := &access.SignupDetails{
			Social: &access.SignupSocial{
				IDPAuth:     idpAuth,
				Provider:    provider,
				RedirectURL: r.Social.RedirectURL,
			},
			Org:       &models.Organization{Name: r.OrgName},
			SubDomain: r.Subdomain,
		}
		created, err = access.Signup(c, keyExpires, a.server.options.BaseDomain, details)
		if err != nil {
			return nil, handleSignupError(err)
		}
	case r.User != nil:
		details := &access.SignupDetails{
			User: &access.SignupUser{
				Name:     r.User.UserName,
				Password: r.User.Password,
			},
			Org:       &models.Organization{Name: r.OrgName},
			SubDomain: r.Subdomain,
		}
		var err error
		created, err = access.Signup(c, keyExpires, a.server.options.BaseDomain, details)
		if err != nil {
			return nil, handleSignupError(err)
		}
	default:
		// make sure to always fail by default
		return nil, fmt.Errorf("%w: missing sign up details", internal.ErrBadRequest)
	}

	/*
		This cookie is set to send on all infra domains, make it expire quickly to prevent an unexpected org being set on requests to other orgs.
		This signup cookie sets the authentication for the next call made to the org and will be exchanged for a long-term auth cookie.
		We have to set this short lived sign-up auth cookie to give the user a valid session on sign-up.
		Since the signup is on the base domain we have to set this cookie there,
		but we want auth cookies to only be sent to their respective orgs so they must be set on their org specific sub-domain after redirect.
	*/
	cookie := cookieConfig{
		Name:    cookieSignupName,
		Value:   created.Bearer,
		Domain:  a.server.options.BaseDomain,
		Expires: time.Now().Add(1 * time.Minute),
	}
	setCookie(c, cookie)

	a.t.User(created.Identity.ID.String(), created.Identity.Name)
	a.t.Org(created.Organization.ID.String(), created.Identity.ID.String(), created.Organization.Name, created.Organization.Domain)
	a.t.Event("signup", created.Identity.ID.String(), created.Organization.ID.String(), Properties{})

	link := fmt.Sprintf("https://%s", created.Organization.Domain)
	err := email.SendSignupEmail("", created.Identity.Name, email.SignupData{
		Link:        link,
		WrappedLink: wrapLinkWithVerification(link, created.Organization.Domain, created.Identity.VerificationToken),
	})
	if err != nil {
		// if email failed, continue on anyway.
		logging.L.Error().Err(err).Msg("could not send signup email")
	}

	return &api.SignupResponse{
		User:         created.Identity.ToAPI(),
		Organization: created.Organization.ToAPI(),
	}, nil
}

// handleSignupError updates internal errors to have the right structure for
// handling by sendAPIError.
func handleSignupError(err error) error {
	var ucErr data.UniqueConstraintError
	if errors.As(err, &ucErr) {
		switch {
		case ucErr.Table == "organizations" && ucErr.Column == "domain":
			// SignupRequest.Org.SubDomain is the field in the request struct.
			apiError := newAPIErrorForUniqueConstraintError(ucErr, err.Error())
			apiError.FieldErrors[0].FieldName = "org.subDomain"
			return apiError
		}
	}
	return err
}

func (a *API) socialSignupUserAuth(c *gin.Context, provider *models.Provider, auth *authn.OIDCAuthn) (*providers.IdentityProviderAuth, error) {
	providerClient, err := a.providerClient(c, provider, auth.RedirectURL)
	if err != nil {
		return nil, fmt.Errorf("sign-up provider client: %w", err)
	}
	auth.OIDCProviderClient = providerClient

	// exchange code for tokens from identity provider (these tokens are for the IDP, not Infra)
	result, err := auth.OIDCProviderClient.ExchangeAuthCodeForProviderTokens(c, auth.Code)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return nil, fmt.Errorf("%w: %s", internal.ErrBadGateway, err.Error())
		}

		logging.L.Debug().Err(err).Msg("failed to exhange sign-up code for tokens")
		return nil, internal.ErrUnauthorized
	}

	return result, nil
}
