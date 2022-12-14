package providers

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"

	"github.com/infrahq/infra/internal/logging"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/internal/validate"
)

const oidcProviderRequestTimeout = time.Second * 10

// UserInfoClaims captures the claims fields from a user-info response that we care about
type UserInfoClaims struct {
	Email  string   `json:"email"` // returned by default for Okta user info
	Groups []string `json:"groups"`
	Name   string   `json:"name"` // returned by default for Azure user info
}

type AuthServerInfo struct {
	AuthURL         string
	ScopesSupported []string `json:"scopes_supported"`
}

type IdentityProviderAuth struct {
	AccessToken       string
	RefreshToken      string
	AccessTokenExpiry time.Time
	Email             string
}

type OIDCClient interface {
	Validate(context.Context) error
	AuthServerInfo(context.Context) (*AuthServerInfo, error)
	ExchangeAuthCodeForProviderTokens(ctx context.Context, code string) (*IdentityProviderAuth, error)
	RefreshAccessToken(ctx context.Context, providerUser *models.ProviderUser) (accessToken string, expiry *time.Time, err error)
	GetUserInfo(ctx context.Context, providerUser *models.ProviderUser) (*UserInfoClaims, error)
}

type key struct{}

var ctxKey = key{}

func OIDCClientFromContext(ctx context.Context) OIDCClient {
	if raw := ctx.Value(ctxKey); raw != nil {
		return raw.(OIDCClient) // nolint:forcetypeassert
	}
	return nil
}

func WithOIDCClient(ctx context.Context, client OIDCClient) context.Context {
	return context.WithValue(ctx, ctxKey, client)
}

type oidcClientImplementation struct {
	ProviderModel models.Provider
	Domain        string
	ClientID      string
	ClientSecret  string
	RedirectURL   string
}

func NewOIDCClient(provider models.Provider, clientSecret, redirectURL string) OIDCClient {
	oidcClient := &oidcClientImplementation{
		Domain:       provider.URL,
		ClientID:     provider.ClientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
	}

	// nolint:exhaustive
	switch provider.Kind {
	case models.ProviderKindAzure:
		return &azure{OIDCClient: oidcClient}
	case models.ProviderKindGoogle:
		return &google{
			OIDCClient: oidcClient,
			GoogleCredentials: googleCredentials{
				PrivateKey:       string(provider.PrivateKey),
				ClientEmail:      provider.ClientEmail,
				DomainAdminEmail: provider.DomainAdminEmail,
			},
		}
	default:
		return oidcClient
	}
}

// Validate tests if an identity provider has valid attributes to support user login
func (o *oidcClientImplementation) Validate(ctx context.Context) error {
	conf, _, err := o.clientConfig(ctx)
	if err != nil {
		logging.Debugf("error validating oidc provider: %s", err)
		return newValidationError("url")
	}

	_, err = conf.Exchange(ctx, "test-code")
	if err != nil {
		var errRetrieve *oauth2.RetrieveError
		if errors.As(err, &errRetrieve) {
			if strings.Contains(string(errRetrieve.Body), "client_id") || strings.Contains(string(errRetrieve.Body), "client id") {
				logging.Debugf("error validating oidc provider client: %s", err)
				return newValidationError("clientID")
			}

			if strings.Contains(string(errRetrieve.Body), "secret") {
				logging.Debugf("error validating oidc provider client: %s", err)
				return newValidationError("clientSecret")
			}
		}
		logging.L.Trace().Err(err).Msg("error validating oidc provider, this is expected")
	}

	// return nil for all other errors, because the request was made with an
	// invalid code, which will always fail
	return nil
}

func newValidationError(field string) error {
	return validate.Error{field: {"invalid provider " + field}}
}

// AuthServerInfo returns details about the oidc server auth URL, and the scopes it supports
func (o *oidcClientImplementation) AuthServerInfo(ctx context.Context) (*AuthServerInfo, error) {
	ctx, cancel := context.WithTimeout(ctx, oidcProviderRequestTimeout)
	defer cancel()
	// find out what the authorization endpoint is
	provider, err := oidc.NewProvider(ctx, fmt.Sprintf("https://%s", o.Domain))
	if err != nil {
		return nil, fmt.Errorf("get provider oidc info: %w", err)
	}

	// claims are the attributes of the user we want to know from the identity provider
	var claims struct {
		ScopesSupported []string `json:"scopes_supported"`
	}

	if err := provider.Claims(&claims); err != nil {
		return nil, fmt.Errorf("could not parse provider claims: %w", err)
	}

	scopes := []string{"openid", "email"} // openid and email are required scopes for login to work

	// we want to be able to use these scopes to access groups, but they are not needed
	wantScope := map[string]bool{
		"groups":         true,
		"offline_access": true,
	}

	for _, scope := range claims.ScopesSupported {
		if wantScope[scope] {
			scopes = append(scopes, scope)
		}
	}

	return &AuthServerInfo{
		AuthURL:         provider.Endpoint().AuthURL,
		ScopesSupported: scopes,
	}, nil
}

// clientConfig returns the OAuth client configuration needed to interact with an identity provider
func (o *oidcClientImplementation) clientConfig(ctx context.Context) (*oauth2.Config, *oidc.Provider, error) {
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
func (o *oidcClientImplementation) tokenSource(ctx context.Context, conf *oauth2.Config, providerTokens *models.ProviderUser) (oauth2.TokenSource, error) {
	userToken := &oauth2.Token{
		AccessToken:  string(providerTokens.AccessToken),
		RefreshToken: string(providerTokens.RefreshToken),
		Expiry:       providerTokens.ExpiresAt,
	}

	return conf.TokenSource(ctx, userToken), nil
}

// ExchangeAuthCodeForProviderTokens exchanges the authorization code a user received on login for valid identity provider tokens
func (o *oidcClientImplementation) ExchangeAuthCodeForProviderTokens(ctx context.Context, code string) (*IdentityProviderAuth, error) {
	ctx, cancel := context.WithTimeout(ctx, oidcProviderRequestTimeout)
	defer cancel()

	conf, provider, err := o.clientConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("client exchange code: %w", err)
	}

	exchanged, err := conf.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("code exchange: %w", err)
	}

	rawAccessToken, ok := exchanged.Extra("access_token").(string)
	if !ok {
		return nil, errors.New("could not extract access token from oauth2")
	}

	rawRefreshToken, ok := exchanged.Extra("refresh_token").(string)
	if !ok {
		// this probably means that the client does not have refresh tokens enabled
		logging.Warnf("no refresh token returned from oidc client for %q, session lifetime will be reduced", o.Domain)
	}

	rawIDToken, ok := exchanged.Extra("id_token").(string)
	if !ok {
		return nil, errors.New("could not extract id_token from oauth2 token")
	}

	// we get sensitive claims from the ID token, must validate them
	verifier := provider.Verifier(&oidc.Config{ClientID: o.ClientID})

	idToken, err := verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return nil, fmt.Errorf("validate id token: %w", err)
	}

	var claims struct {
		Email string `json:"email"`
	}

	if err := idToken.Claims(&claims); err != nil {
		return nil, fmt.Errorf("id token claims: %w", err)
	}

	if claims.Email == "" {
		err := fmt.Errorf("ID token claim is missing an email address")
		return nil, err
	}

	if strings.ContainsAny(claims.Email, ` '`) {
		err := fmt.Errorf("ID token claim has invalid email address")
		return nil, err
	}

	return &IdentityProviderAuth{
		AccessToken:       rawAccessToken,
		RefreshToken:      rawRefreshToken,
		AccessTokenExpiry: exchanged.Expiry,
		Email:             claims.Email,
	}, nil
}

// RefreshAccessToken uses the refresh token to get a new access token if it is expired
func (o *oidcClientImplementation) RefreshAccessToken(ctx context.Context, providerUser *models.ProviderUser) (accessToken string, expiry *time.Time, err error) {
	ctx, cancel := context.WithTimeout(ctx, oidcProviderRequestTimeout)
	defer cancel()

	conf, _, err := o.clientConfig(ctx)
	if err != nil {
		return "", nil, fmt.Errorf("call idp with tokens: %w", err)
	}

	tokenSource, err := o.tokenSource(ctx, conf, providerUser)
	if err != nil {
		return "", nil, fmt.Errorf("ref token source: %w", err)
	}

	newToken, err := tokenSource.Token() // this refreshes token if needed
	if err != nil {
		return "", nil, fmt.Errorf("refresh user token: %w", err)
	}

	return newToken.AccessToken, &newToken.Expiry, nil
}

// GetUserInfo uses a provider token to call the OpenID Connect UserInfo endpoint,
// make sure an access token is valid (not expired) before using this
func (o *oidcClientImplementation) GetUserInfo(ctx context.Context, providerUser *models.ProviderUser) (*UserInfoClaims, error) {
	ctx, cancel := context.WithTimeout(ctx, oidcProviderRequestTimeout)
	defer cancel()

	conf, provider, err := o.clientConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("user info client: %w", err)
	}

	tokenSource, err := o.tokenSource(ctx, conf, providerUser)
	if err != nil {
		return nil, fmt.Errorf("info token source: %w", err)
	}

	info, err := provider.UserInfo(ctx, tokenSource)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return nil, err
		}
		return nil, fmt.Errorf("get user info: %w", err)
	}

	claims := &UserInfoClaims{}
	if err := info.Claims(claims); err != nil {
		return nil, fmt.Errorf("user info claims: %w", err)
	}

	if claims.Name == "" && claims.Email == "" {
		return nil, fmt.Errorf("claim must include either a name or email")
	}

	return claims, nil
}
