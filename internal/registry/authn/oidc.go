package authn

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/registry/models"
	"golang.org/x/oauth2"
	"gopkg.in/square/go-jose.v2/jwt"
)

// UserInfo captures the fields from a user-info response that we care about
type UserInfo struct {
	Email  string   `json:"email"`
	Groups []string `json:"groups"`
}

type OIDC interface {
	ExchangeAuthCodeForProviderTokens(code string) (accessToken, refreshToken string, accessTokenExpiry time.Time, email string, err error)
	RefreshAccessToken(providerToken *models.ProviderToken) (accessToken string, expiry *time.Time, err error)
	GetUserInfo(providerToken *models.ProviderToken) (*UserInfo, error)
}

type oidcImplementation struct {
	Domain       string
	ClientID     string
	ClientSecret string
}

func NewOIDC(domain, clientID, clientSecret string) OIDC {
	return &oidcImplementation{
		Domain:       domain,
		ClientID:     clientID,
		ClientSecret: clientSecret,
	}
}

// clientConfig returns the OAuth client configuration needed to interact with an identity provider
func (o *oidcImplementation) clientConfig() *oauth2.Config {
	return &oauth2.Config{
		ClientID:     o.ClientID,
		ClientSecret: o.ClientSecret,
		RedirectURL:  "http://localhost:8301",
		Scopes:       []string{"openid", "email", "groups", "offline_access"},
		Endpoint: oauth2.Endpoint{
			TokenURL: fmt.Sprintf("https://%s/oauth2/v1/token", o.Domain),
			AuthURL:  fmt.Sprintf("https://%s/oauth2/v1/authorize", o.Domain),
		},
	}
}

// tokenSource is used to call an identity provider with the specified provider tokens
func (o *oidcImplementation) tokenSource(providerTokens *models.ProviderToken) oauth2.TokenSource {
	ctx := context.Background()
	conf := o.clientConfig()

	userToken := &oauth2.Token{
		AccessToken:  string(providerTokens.AccessToken),
		RefreshToken: string(providerTokens.RefreshToken),
		Expiry:       providerTokens.Expiry,
	}

	return conf.TokenSource(ctx, userToken)
}

func (o *oidcImplementation) ExchangeAuthCodeForProviderTokens(code string) (accessToken, refreshToken string, accessTokenExpiry time.Time, email string, err error) {
	ctx := context.Background()
	conf := o.clientConfig()

	exchanged, err := conf.Exchange(ctx, code)
	if err != nil {
		return "", "", time.Time{}, "", fmt.Errorf("code exchange: %w", err)
	}

	accessToken, ok := exchanged.Extra("access_token").(string)
	if !ok {
		return "", "", time.Time{}, "", errors.New("could not extract access token from oauth2")
	}

	refreshToken, ok = exchanged.Extra("refresh_token").(string)
	if !ok {
		return "", "", time.Time{}, "", errors.New("could not extract refresh token from oauth2")
	}

	idToken, ok := exchanged.Extra("id_token").(string)
	if !ok {
		return "", "", time.Time{}, "", errors.New("could not extract id_token from oauth2 token")
	}

	parsedAcc, err := jwt.ParseSigned(accessToken)
	if err != nil {
		return "", "", time.Time{}, "", fmt.Errorf("validate acc: %w", err)
	}

	parsedID, err := jwt.ParseSigned(idToken)
	if err != nil {
		return "", "", time.Time{}, "", fmt.Errorf("parse signed: %w", err)
	}

	accClaims := &jwt.Claims{}

	// TODO: #815, validate claims
	err = parsedAcc.UnsafeClaimsWithoutVerification(accClaims)
	if err != nil {
		return "", "", time.Time{}, "", fmt.Errorf("acc token claims: %w", err)
	}

	attributes := make(map[string]interface{})
	if err := parsedID.UnsafeClaimsWithoutVerification(&attributes); err != nil {
		return "", "", time.Time{}, "", fmt.Errorf("id cliams: %w", err)
	}

	email, ok = attributes["email"].(string)
	if !ok {
		return "", "", time.Time{}, "", errors.New("could not extract email from identity provider token")
	}

	return accessToken, refreshToken, accClaims.Expiry.Time(), email, nil
}

// RefreshAccessToken uses the refresh token to get a new access token if it is expired
func (o *oidcImplementation) RefreshAccessToken(providerTokens *models.ProviderToken) (accessToken string, expiry *time.Time, err error) {
	tokenSource := o.tokenSource(providerTokens)

	newToken, err := tokenSource.Token() // this refreshes token if needed
	if err != nil {
		return "", nil, fmt.Errorf("refresh user token: %w", err)
	}

	return newToken.AccessToken, &newToken.Expiry, nil
}

// GetUserInfo uses a provider token to get the current information about a user,
// make sure an access token is valid (not expired) before using this
func (o *oidcImplementation) GetUserInfo(providerTokens *models.ProviderToken) (*UserInfo, error) {
	ctx := context.Background()
	tokenSource := o.tokenSource(providerTokens)

	userInfoEndpoint := fmt.Sprintf("https://%s/oauth2/v1/userinfo", o.Domain)

	client := oauth2.NewClient(ctx, tokenSource)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, userInfoEndpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("userinfo request %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+string(providerTokens.AccessToken))

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("user info response: %w", err)
	}

	if resp.StatusCode == http.StatusUnauthorized {
		// access token has been revoked, user is no longer valid
		return nil, internal.ErrForbidden
	}

	info := &UserInfo{}

	err = json.NewDecoder(resp.Body).Decode(info)
	if err != nil {
		return nil, fmt.Errorf("decode user info response: %w", err)
	}

	return info, nil
}
