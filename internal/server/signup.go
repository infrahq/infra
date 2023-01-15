package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

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
	rCtx := getRequestContext(c)
	if !a.server.options.EnableSignup {
		return nil, fmt.Errorf("%w: signup is disabled", internal.ErrBadRequest)
	}

	keyExpires := time.Now().UTC().Add(a.server.options.SessionDuration)

	var created *NewOrgDetails
	switch {
	case r.Social != nil:
		// do social sign-up
		if a.server.Google == nil {
			return nil, fmt.Errorf("%w: google login is not configured", internal.ErrBadRequest)
		}
		// check if an org exists with their desired sub-domain
		// this has to be done here since the auth code is single-use
		if err := access.DomainAvailable(rCtx, fmt.Sprintf("%s.%s", r.Subdomain, a.server.options.BaseDomain)); err != nil {
			return nil, err
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
		details := SignupDetails{
			Social: &SignupSocial{
				IDPAuth:     idpAuth,
				Provider:    provider,
				RedirectURL: r.Social.RedirectURL,
			},
			Org:       &models.Organization{Name: r.OrgName},
			SubDomain: r.Subdomain,
		}
		created, err = createOrgAndUserForSignup(c, keyExpires, a.server.options.BaseDomain, details)
		if err != nil {
			return nil, handleSignupError(err)
		}
	case r.User != nil:
		details := SignupDetails{
			User: &SignupUser{
				Name:     r.User.UserName,
				Password: r.User.Password,
			},
			Org:       &models.Organization{Name: r.OrgName},
			SubDomain: r.Subdomain,
		}
		var err error
		created, err = createOrgAndUserForSignup(c, keyExpires, a.server.options.BaseDomain, details)
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
	setCookie(c.Request, rCtx.Response.HTTPWriter, cookie)

	a.t.User(created.Identity.ID.String(), created.Identity.Name)
	a.t.Org(created.Organization.ID.String(), created.Identity.ID.String(), created.Organization.Name, created.Organization.Domain)
	a.t.Event("signup", created.Identity.ID.String(), created.Organization.ID.String(), Properties{}.Set("email", created.Identity.Name))

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
	providerClient, err := a.server.providerClient(c.Request.Context(), provider, auth.RedirectURL)
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

// NewOrgDetails are details about the identity, org, and access key after a sign-up is successful
type NewOrgDetails struct {
	Identity     *models.Identity
	Organization *models.Organization
	Bearer       string
}

// SocialSignup stores the information about a sign-up from a successful OIDC authentication
type SignupSocial struct {
	IDPAuth     *providers.IdentityProviderAuth
	RedirectURL string // stored on provider user to use refresh token in the future
	Provider    *models.Provider
}

// SocialUser allows a user to sign-up with an email and a password
type SignupUser struct {
	Name     string
	Password string
}

type SignupDetails struct {
	User      *SignupUser
	Social    *SignupSocial
	Org       *models.Organization
	SubDomain string
}

// createOrgAndUserForSignup creates a user identity using the supplied name and password and
// grants the identity "admin" access to Infra.
func createOrgAndUserForSignup(c *gin.Context, keyExpiresAt time.Time, baseDomain string, details SignupDetails) (*NewOrgDetails, error) {
	if details.Social == nil && details.User == nil {
		return nil, fmt.Errorf("sign-up requires social login details or user details")
	}

	rCtx := getRequestContext(c)
	db := rCtx.DBTxn

	details.Org.Domain = sanitizedDomain(details.SubDomain, baseDomain)

	adminEmail := ""
	switch {
	case details.User != nil:
		adminEmail = details.User.Name
	case details.Social != nil:
		adminEmail = details.Social.IDPAuth.Email
	}
	allowedLoginDomain, err := email.Domain(adminEmail)
	if err != nil {
		return nil, fmt.Errorf("allowed login domain from admin email: %w", err)
	}
	if allowedLoginDomain != "gmail.com" && allowedLoginDomain != "googlemail.com" {
		// if gmail or googlemail the admin will have to manually specify this later
		details.Org.AllowedDomains = []string{allowedLoginDomain}
	}

	if err := data.CreateOrganization(db, details.Org); err != nil {
		return nil, fmt.Errorf("create org on sign-up: %w", err)
	}

	db = db.WithOrgID(details.Org.ID)
	rCtx.DBTxn = db
	rCtx.Authenticated.Organization = details.Org
	c.Set(access.RequestContextKey, rCtx)

	var identity *models.Identity
	bearer := ""
	switch {
	case details.User != nil:
		// username/password sign-up
		user := &models.ProviderUser{
			ProviderID: data.InfraProvider(db).ID,
			Email:      details.User.Name,
			LastUpdate: time.Now().UTC(),
			Active:     true,
		}

		var err error
		identity, bearer, err = signupUser(c, keyExpiresAt, user)
		if err != nil {
			return nil, err
		}

		hash, err := bcrypt.GenerateFromPassword([]byte(details.User.Password), bcrypt.DefaultCost)
		if err != nil {
			return nil, fmt.Errorf("hash password on sign-up: %w", err)
		}

		credential := &models.Credential{
			IdentityID:   identity.ID,
			PasswordHash: hash,
		}

		if err := data.CreateCredential(db, credential); err != nil {
			return nil, fmt.Errorf("create credential on sign-up: %w", err)
		}
	case details.Social != nil:
		// sign-up with social (google)
		user := &models.ProviderUser{
			ProviderID:   models.InternalGoogleProviderID,
			Email:        details.Social.IDPAuth.Email,
			RedirectURL:  details.Social.RedirectURL,
			AccessToken:  models.EncryptedAtRest(details.Social.IDPAuth.AccessToken),
			RefreshToken: models.EncryptedAtRest(details.Social.IDPAuth.RefreshToken),
			ExpiresAt:    details.Social.IDPAuth.AccessTokenExpiry,
			LastUpdate:   time.Now().UTC(),
			Active:       true,
		}

		var err error
		identity, bearer, err = signupUser(c, keyExpiresAt, user)
		if err != nil {
			return nil, err
		}
	default:
		// this should have been caught by the initial error check
		return nil, fmt.Errorf("sign-up requires social login or user credentials")
	}

	return &NewOrgDetails{
		Identity:     identity,
		Organization: details.Org,
		Bearer:       bearer,
	}, nil
}

// signupUser creates the user identity and grants for a new org
func signupUser(c *gin.Context, keyExpiresAt time.Time, user *models.ProviderUser) (*models.Identity, string, error) {
	rCtx := getRequestContext(c)
	tx := rCtx.DBTxn

	identity := &models.Identity{
		Name: user.Email,
	}
	if err := data.CreateIdentity(tx, identity); err != nil {
		return nil, "", fmt.Errorf("create identity on sign-up: %w", err)
	}
	user.IdentityID = identity.ID

	// create the provider user with the specified fields
	err := data.ProvisionProviderUser(tx, user)
	if err != nil {
		return nil, "", fmt.Errorf("create provider user on sign-up: %w", err)
	}

	err = data.CreateGrant(tx, &models.Grant{
		Subject:   models.NewSubjectForUser(identity.ID),
		Privilege: models.InfraAdminRole,
		Resource:  access.ResourceInfraAPI,
		CreatedBy: identity.ID,
	})
	if err != nil {
		return nil, "", fmt.Errorf("create grant on sign-up: %w", err)
	}

	// grant the user a session on initial sign-up
	accessKey := &models.AccessKey{
		IssuedFor:     identity.ID,
		IssuedForName: identity.Name,
		ProviderID:    user.ProviderID,
		ExpiresAt:     keyExpiresAt,
		Scopes:        []string{models.ScopeAllowCreateAccessKey},
	}

	bearer, err := data.CreateAccessKey(tx, accessKey)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create access key after sign-up: %w", err)
	}

	// Update the request context so that logging middleware can include the userID
	rCtx.Authenticated.User = identity
	c.Set(access.RequestContextKey, rCtx)

	return identity, bearer, nil
}

func sanitizedDomain(subDomain, serverBaseDomain string) string {
	return strings.ToLower(subDomain) + "." + serverBaseDomain
}

// See docs/dev/api-versioned-handlers.md for a guide to adding new version handlers.
func (a *API) addPreviousVersionHandlersSignup() {
	type signupOrgV0_19_0 struct {
		Name      string `json:"name"`
		Subdomain string `json:"subDomain"`
	}
	type signupRequestV0_19_0 struct {
		Name     string           `json:"name"`
		Password string           `json:"password"`
		Org      signupOrgV0_19_0 `json:"org"`
	}

	addVersionHandler(a,
		http.MethodPost, "/api/signup", "0.19.0",
		route[signupRequestV0_19_0, *api.SignupResponse]{
			handler: func(c *gin.Context, reqOld *signupRequestV0_19_0) (*api.SignupResponse, error) {
				req := &api.SignupRequest{
					User: &api.SignupUser{
						UserName: reqOld.Name,
						Password: reqOld.Password,
					},
					OrgName:   reqOld.Org.Name,
					Subdomain: reqOld.Org.Subdomain,
				}
				return a.Signup(c, req)
			},
		},
	)
}
