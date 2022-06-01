package authn

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
	"gorm.io/gorm"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/logging"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/uid"
)

const oidcProviderRequestTimeout = time.Second * 10

// InfoClaims captures the claims fields from a user-info response that we care about
type InfoClaims struct {
	Email  string   `json:"email"`
	Groups []string `json:"groups"`
}

type oidcAuthn struct {
	ProviderID         uid.ID
	RedirectURL        string
	Code               string
	OIDCProviderClient OIDC
}

func NewOIDCAuthentication(providerID uid.ID, redirectURL string, code string, oidcProviderClient OIDC) LoginMethod {
	return &oidcAuthn{
		ProviderID:         providerID,
		RedirectURL:        redirectURL,
		Code:               code,
		OIDCProviderClient: oidcProviderClient,
	}
}

type OIDC interface {
	Validate() error
	ExchangeAuthCodeForProviderTokens(code string) (accessToken, refreshToken string, accessTokenExpiry time.Time, email string, err error)
	RefreshAccessToken(providerUser *models.ProviderUser) (accessToken string, expiry *time.Time, err error)
	GetUserInfo(providerUser *models.ProviderUser) (*InfoClaims, error)
}

type oidcImplementation struct {
	Domain       string
	ClientID     string
	ClientSecret string
	RedirectURL  string
}

func NewOIDC(domain, clientID, clientSecret, redirectURL string) OIDC {
	return &oidcImplementation{
		Domain:       domain,
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
	}
}

func (a *oidcAuthn) Authenticate(db *gorm.DB) (*models.Identity, *models.Provider, error) {
	provider, err := data.GetProvider(db, data.ByID(a.ProviderID))
	if err != nil {
		return nil, nil, err
	}

	// exchange code for tokens from identity provider (these tokens are for the IDP, not Infra)
	accessToken, refreshToken, expiry, email, err := a.OIDCProviderClient.ExchangeAuthCodeForProviderTokens(a.Code)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return nil, nil, fmt.Errorf("%w: %s", internal.ErrBadGateway, err.Error())
		}

		return nil, nil, fmt.Errorf("exhange code for tokens: %w", err)
	}

	identity, err := data.GetIdentity(db.Preload("Groups"), data.ByName(email))
	if err != nil {
		if !errors.Is(err, internal.ErrNotFound) {
			return nil, nil, fmt.Errorf("get user: %w", err)
		}

		identity = &models.Identity{Name: email}

		if err := data.CreateIdentity(db, identity); err != nil {
			return nil, nil, fmt.Errorf("create user: %w", err)
		}
	}

	providerUser, err := data.CreateProviderUser(db, provider, identity)
	if err != nil {
		return nil, nil, fmt.Errorf("add user for provider login: %w", err)
	}

	providerUser.RedirectURL = a.RedirectURL
	providerUser.AccessToken = models.EncryptedAtRest(accessToken)
	providerUser.RefreshToken = models.EncryptedAtRest(refreshToken)
	providerUser.ExpiresAt = expiry
	err = data.UpdateProviderUser(db, providerUser)
	if err != nil {
		return nil, nil, fmt.Errorf("UpdateProviderUser: %w", err)
	}

	// get current identity provider groups
	info, err := a.OIDCProviderClient.GetUserInfo(providerUser)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return nil, nil, fmt.Errorf("%w: %s", internal.ErrBadGateway, err.Error())
		}

		return nil, nil, fmt.Errorf("login user info: %w", err)
	}

	err = UpdateUserInfoFromProvider(db, info, identity, provider)
	if err != nil {
		return nil, nil, fmt.Errorf("update info on login: %w", err)
	}

	return identity, provider, nil
}

func (a *oidcAuthn) Name() string {
	return "oidc"
}

func (a *oidcAuthn) RequiresUpdate(db *gorm.DB) (bool, error) {
	return false, nil // not applicable to oidc
}

// Validate tests if an identity provider has valid attributes to support user login
func (o *oidcImplementation) Validate() error {
	ctx := context.Background()
	conf, _, err := o.clientConfig(ctx)
	if err != nil {
		logging.S.Debugf("error validating oidc provider: %s", err)
		return ErrInvalidProviderURL
	}

	_, err = conf.Exchange(ctx, "test-code") // 'test-code' is a placeholder for a valid authorization code, it will always fail
	if err != nil {
		var errRetrieve *oauth2.RetrieveError
		if errors.As(err, &errRetrieve) {
			if strings.Contains(string(errRetrieve.Body), "client_id") || strings.Contains(string(errRetrieve.Body), "client id") {
				logging.S.Debugf("error validating oidc provider client: %s", err)
				return ErrInvalidProviderClientID
			}

			if strings.Contains(string(errRetrieve.Body), "secret") {
				logging.S.Debugf("error validating oidc provider client: %s", err)
				return ErrInvalidProviderClientSecret
			}
		}
		logging.S.Debug(err)
	}

	return nil
}

// clientConfig returns the OAuth client configuration needed to interact with an identity provider
func (o *oidcImplementation) clientConfig(ctx context.Context) (*oauth2.Config, *oidc.Provider, error) {
	provider, err := oidc.NewProvider(ctx, fmt.Sprintf("https://%s", o.Domain))
	if err != nil {
		return nil, nil, fmt.Errorf("get provider openid info: %w", err)
	}

	conf := &oauth2.Config{
		ClientID:     o.ClientID,
		ClientSecret: o.ClientSecret,
		RedirectURL:  o.RedirectURL,
		Scopes:       []string{oidc.ScopeOpenID, "email", "groups", oidc.ScopeOfflineAccess},
		Endpoint:     provider.Endpoint(),
	}

	return conf, provider, nil
}

// tokenSource is used to call an identity provider with the specified provider tokens
func (o *oidcImplementation) tokenSource(providerTokens *models.ProviderUser) (oauth2.TokenSource, error) {
	ctx := context.Background()

	conf, _, err := o.clientConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("call idp with tokens: %w", err)
	}

	userToken := &oauth2.Token{
		AccessToken:  string(providerTokens.AccessToken),
		RefreshToken: string(providerTokens.RefreshToken),
		Expiry:       providerTokens.ExpiresAt,
	}

	return conf.TokenSource(ctx, userToken), nil
}

func (o *oidcImplementation) ExchangeAuthCodeForProviderTokens(code string) (rawAccessToken, rawRefreshToken string, accessTokenExpiry time.Time, email string, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), oidcProviderRequestTimeout)
	defer cancel()

	conf, provider, err := o.clientConfig(ctx)
	if err != nil {
		return "", "", time.Time{}, "", fmt.Errorf("client exchange code: %w", err)
	}

	exchanged, err := conf.Exchange(ctx, code)
	if err != nil {
		return "", "", time.Time{}, "", fmt.Errorf("code exchange: %w", err)
	}

	rawAccessToken, ok := exchanged.Extra("access_token").(string)
	if !ok {
		return "", "", time.Time{}, "", errors.New("could not extract access token from oauth2")
	}

	rawRefreshToken, ok = exchanged.Extra("refresh_token").(string)
	if !ok {
		// this probably means that the client does not have refresh tokens enabled
		logging.S.Warnf("no refresh token returned from oidc client for %q, session lifetime will be reduced", o.Domain)
	}

	rawIDToken, ok := exchanged.Extra("id_token").(string)
	if !ok {
		return "", "", time.Time{}, "", errors.New("could not extract id_token from oauth2 token")
	}

	// we get sensitive claims from the ID token, must validate them
	verifier := provider.Verifier(&oidc.Config{ClientID: o.ClientID})

	idToken, err := verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return "", "", time.Time{}, "", fmt.Errorf("validate id token: %w", err)
	}

	var claims struct {
		Email string `json:"email"`
	}

	if err := idToken.Claims(&claims); err != nil {
		return "", "", time.Time{}, "", fmt.Errorf("id claims: %w", err)
	}

	return rawAccessToken, rawRefreshToken, exchanged.Expiry, claims.Email, nil
}

// RefreshAccessToken uses the refresh token to get a new access token if it is expired
func (o *oidcImplementation) RefreshAccessToken(providerUser *models.ProviderUser) (accessToken string, expiry *time.Time, err error) {
	tokenSource, err := o.tokenSource(providerUser)
	if err != nil {
		return "", nil, fmt.Errorf("ref token source: %w", err)
	}

	newToken, err := tokenSource.Token() // this refreshes token if needed
	if err != nil {
		return "", nil, fmt.Errorf("refresh user token: %w", err)
	}

	return newToken.AccessToken, &newToken.Expiry, nil
}

// GetUserInfo uses a provider token to get the current information about a user,
// make sure an access token is valid (not expired) before using this
func (o *oidcImplementation) GetUserInfo(providerUser *models.ProviderUser) (*InfoClaims, error) {
	ctx, cancel := context.WithTimeout(context.Background(), oidcProviderRequestTimeout)
	defer cancel()

	tokenSource, err := o.tokenSource(providerUser)
	if err != nil {
		return nil, fmt.Errorf("info token source: %w", err)
	}

	_, provider, err := o.clientConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("user info client: %w", err)
	}

	info, err := provider.UserInfo(ctx, tokenSource)
	if err != nil {
		return nil, fmt.Errorf("get user info: %w", err)
	}

	claims := &InfoClaims{}
	if err := info.Claims(claims); err != nil {
		return nil, fmt.Errorf("user info claims: %w", err)
	}

	return claims, nil
}

// UpdateUserInfoFromProvider calls the user info endpoint of an external identity provider to see a user's current attributes
func UpdateUserInfoFromProvider(db *gorm.DB, info *InfoClaims, user *models.Identity, provider *models.Provider) error {
	// add user to groups they are currently in
	var groups []string

	for i := range info.Groups {
		name := info.Groups[i]
		groups = append(groups, name)
	}

	logging.S.Debugf("%s user authenticated with %q groups", provider.Name, groups)

	if err := data.AssignIdentityToGroups(db, user, provider, groups); err != nil {
		return fmt.Errorf("assign identity to groups: %w", err)
	}

	return nil
}
