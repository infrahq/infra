package providers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	googleOAuth "golang.org/x/oauth2/google"
	googleAdminDirectory "google.golang.org/api/admin/directory/v1"
	googleOption "google.golang.org/api/option"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/logging"
	"github.com/infrahq/infra/internal/server/models"
)

type testGroupsKey struct{}

var ErrGoogleClientNotConfigured = fmt.Errorf("google provider api client not configured")

var googleAPIScopes = []string{"https://www.googleapis.com/auth/admin.directory.group.readonly"}

type googleCredentials struct {
	PrivateKey  string
	ClientEmail string
	DomainAdmin string
}

type google struct {
	GoogleCredentials googleCredentials
	OIDCClient        OIDCClient
}

func (g *google) Validate(ctx context.Context) error {
	return g.OIDCClient.Validate(ctx)
}

func (g *google) AuthServerInfo(ctx context.Context) (*AuthServerInfo, error) {
	return g.OIDCClient.AuthServerInfo(ctx)
}

func (g *google) ExchangeAuthCodeForProviderTokens(ctx context.Context, code string) (rawAccessToken, rawRefreshToken string, accessTokenExpiry time.Time, email string, err error) {
	return g.OIDCClient.ExchangeAuthCodeForProviderTokens(ctx, code)
}

func (g *google) RefreshAccessToken(ctx context.Context, providerUser *models.ProviderUser) (accessToken string, expiry *time.Time, err error) {
	return g.OIDCClient.RefreshAccessToken(ctx, providerUser)
}

func (g *google) GetUserInfo(ctx context.Context, providerUser *models.ProviderUser) (*UserInfoClaims, error) {
	// this checks if the user still exists
	info, err := g.OIDCClient.GetUserInfo(ctx, providerUser)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return nil, fmt.Errorf("%w: %s", internal.ErrBadGateway, err.Error())
		}
		return nil, fmt.Errorf("could not get user info from provider: %w", err)
	}

	newGroups, err := g.checkGoogleWorkspaceGroups(ctx, providerUser)
	if err != nil {
		logging.Debugf("unable to retrieve groups from google api: %s", err.Error())

		if errors.Is(err, context.DeadlineExceeded) {
			return nil, fmt.Errorf("%w: %s", internal.ErrBadGateway, err.Error())
		}

		// these errors just mean that the groups API client was not configured, we can continue
		if !errors.Is(err, ErrUnauthorized) && !errors.Is(err, ErrGoogleClientNotConfigured) {
			return nil, fmt.Errorf("could not check google user groups: %w", err)
		}

		newGroups = []string{} // set the groups empty to clear them
		logging.Warnf("Unable to get groups from the Google API for provider ID:%q. Make sure the service account has the required permissions.", providerUser.ProviderID)
	}

	info.Groups = newGroups

	logging.Debugf("user synchronized with %q groups from google provider", &newGroups)

	return info, nil
}

func (g *google) checkGoogleWorkspaceGroups(ctx context.Context, providerUser *models.ProviderUser) ([]string, error) {
	if g.GoogleCredentials.ClientEmail == "" || g.GoogleCredentials.DomainAdmin == "" || g.GoogleCredentials.PrivateKey == "" {
		// not configured for groups, skip
		return []string{}, ErrGoogleClientNotConfigured
	}

	params := googleOAuth.CredentialsParams{
		Scopes:  googleAPIScopes,
		Subject: g.GoogleCredentials.DomainAdmin, // delegated admin permissions scopes to the groups endpoint are required by the client
	}

	// credentialsFile emulates the format of a Google credentials file
	type credentialsFile struct {
		Type        string `json:"type"`
		PrivateKey  string `json:"private_key"`
		ClientEmail string `json:"client_email"`
		AuthURI     string `json:"auth_uri"`
		TokenURI    string `json:"token_uri"`
	}

	credsFile := credentialsFile{
		Type:        "service_account",
		PrivateKey:  g.GoogleCredentials.PrivateKey,
		ClientEmail: g.GoogleCredentials.ClientEmail,
		AuthURI:     "https://accounts.google.com/o/oauth2/auth",
		TokenURI:    "https://oauth2.googleapis.com/token",
	}

	credBytes, err := json.Marshal(credsFile)
	if err != nil {
		return []string{}, fmt.Errorf("failed to marshal google credentials: %w", err)
	}

	creds, err := googleOAuth.CredentialsFromJSONWithParams(ctx, credBytes, params)
	if err != nil {
		return []string{}, fmt.Errorf("unable to create google credentials: %w", err)
	}

	client, err := googleAdminDirectory.NewService(ctx, googleOption.WithCredentials(creds))
	if err != nil {
		return []string{}, fmt.Errorf("unable to create google admin directory service: %w", err)
	}

	if val := ctx.Value(testGroupsKey{}); val != nil {
		// stub out the external call for unit tests
		testGroups, ok := val.([]string)
		if !ok {
			return []string{}, fmt.Errorf("failed to parse test groups")
		}
		return testGroups, nil
	}

	resp, err := client.Groups.List().UserKey(providerUser.Email).Do()
	if err != nil {
		return []string{}, fmt.Errorf("unable to call google workspace directory api: %w", err)
	}

	groupNames := []string{}
	for _, g := range resp.Groups {
		groupNames = append(groupNames, g.Name)
	}

	return groupNames, nil
}
