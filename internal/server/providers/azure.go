package providers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"golang.org/x/oauth2"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/logging"
	"github.com/infrahq/infra/internal/server/models"
)

const graphGroupDataType = "#microsoft.graph.group"

// this should only be changed for tests
var graphGroupMemberEndpoint = "https://graph.microsoft.com/v1.0/me/memberOf"

type graphObject struct {
	Type        string `json:"@odata.type"`
	DisplayName string `json:"displayName"`
}

type graphResponse struct {
	Context string        `json:"@odata.context"`
	Value   []graphObject `json:"value"`
}

type azure struct {
	OIDC OIDC
}

func (a *azure) Validate(ctx context.Context) error {
	return a.OIDC.Validate(ctx)
}

func (a *azure) AuthServerInfo(ctx context.Context) (*AuthServerInfo, error) {
	return a.OIDC.AuthServerInfo(ctx)
}

func (a *azure) ExchangeAuthCodeForProviderTokens(ctx context.Context, code string) (rawAccessToken, rawRefreshToken string, accessTokenExpiry time.Time, email string, err error) {
	return a.OIDC.ExchangeAuthCodeForProviderTokens(ctx, code)
}

func (a *azure) RefreshAccessToken(ctx context.Context, providerUser *models.ProviderUser) (accessToken string, expiry *time.Time, err error) {
	return a.OIDC.RefreshAccessToken(ctx, providerUser)
}

func (a *azure) GetUserInfo(ctx context.Context, providerUser *models.ProviderUser) (*UserInfoClaims, error) {
	// this checks if the user still exists
	info, err := a.OIDC.GetUserInfo(ctx, providerUser)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return nil, fmt.Errorf("%w: %s", internal.ErrBadGateway, err.Error())
		}
		return nil, fmt.Errorf("could not get user info from provider: %w", err)
	}

	newGroups, err := checkMemberOfGraphGroups(ctx, string(providerUser.AccessToken))
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return nil, fmt.Errorf("%w: %s", internal.ErrBadGateway, err.Error())
		}

		if !errors.Is(err, ErrUnauthorized) {
			return nil, fmt.Errorf("could not check azure user groups: %w", err)
		}

		newGroups = []string{} // set the groups empty to clear them
		logging.Warnf("Unable to get groups from the Azure API for %q provider. Make sure the application client has the required permissions.", provider.Name)
	}

	info.Groups = newGroups

	logging.Debugf("user synchronized with %q groups from azure provider", &newGroups)

	return info, nil
}

// checkMemberOfGraphGroups calls the Microsoft Graph API to find out what groups a user belongs to
func checkMemberOfGraphGroups(ctx context.Context, accessToken string) ([]string, error) {
	bearer := "Bearer " + accessToken

	req, err := http.NewRequest(http.MethodGet, graphGroupMemberEndpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create azure groups request: %w", err)
	}

	req.Header.Add("Authorization", bearer)

	ctx, cancel := context.WithTimeout(ctx, oidcProviderRequestTimeout)
	defer cancel()

	client := http.DefaultClient
	if c, ok := ctx.Value(oauth2.HTTPClient).(*http.Client); ok {
		client = c // used in tests for specific transport needs, like skipping TLS verify
	}
	resp, err := client.Do(req.WithContext(ctx))
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return nil, fmt.Errorf("%w: %s", internal.ErrBadGateway, err.Error())
		}
		return nil, fmt.Errorf("failed to query azure for groups: %w", err)
	}

	if resp.StatusCode == http.StatusForbidden {
		return nil, ErrUnauthorized
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("could not read azure groups response: %w", err)
	}

	graphResp := graphResponse{}
	err = json.Unmarshal(body, &graphResp)
	if err != nil {
		return nil, fmt.Errorf("could not parse azure groups response: %w", err)
	}

	groups := []string{}
	for _, object := range graphResp.Value {
		if object.Type == graphGroupDataType {
			groups = append(groups, object.DisplayName)
		}
	}

	return groups, nil
}
