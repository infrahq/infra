package providers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"gorm.io/gorm"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/logging"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
)

const (
	graphGroupMemberEndpoint = "https://graph.microsoft.com/v1.0/me/memberOf"
	graphGroupDataType       = "#microsoft.graph.group"
)

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

func (a *azure) Validate() error {
	return a.OIDC.Validate()
}

func (a *azure) ExchangeAuthCodeForProviderTokens(code string) (rawAccessToken, rawRefreshToken string, accessTokenExpiry time.Time, email string, err error) {
	return a.OIDC.ExchangeAuthCodeForProviderTokens(code)
}

func (a *azure) RefreshAccessToken(providerUser *models.ProviderUser) (accessToken string, expiry *time.Time, err error) {
	return a.OIDC.RefreshAccessToken(providerUser)
}

func (a *azure) GetUserInfo(providerUser *models.ProviderUser) (*InfoClaims, error) {
	return a.OIDC.GetUserInfo(providerUser)
}

func (a *azure) SyncProviderUser(db *gorm.DB, user *models.Identity, provider *models.Provider) error {
	providerUser, err := data.GetProviderUser(db, provider.ID, user.ID)
	if err != nil {
		return err
	}

	if err := checkRefreshAccessToken(db, providerUser, a); err != nil {
		return fmt.Errorf("oidc sync failed to check users access token: %w", err)
	}

	// this checks if the user still exists
	_, err = a.OIDC.GetUserInfo(providerUser)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return fmt.Errorf("%w: %s", internal.ErrBadGateway, err.Error())
		}
		return fmt.Errorf("could not get user info from provider: %w", err)
	}

	newGroups, err := checkMemberOfGraphGroups(string(providerUser.AccessToken))
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return fmt.Errorf("%w: %s", internal.ErrBadGateway, err.Error())
		} else if errors.Is(err, ErrUnauthorized) {
			newGroups = &[]string{} // set the groups empty to clear them
			logging.S.Warnf("Unable to get groups from the Azure API for %q provider. Make sure the application client has the required permissions.", provider.Name)
		} else {
			return fmt.Errorf("could not check azure user groups: %w", err)
		}
	}

	logging.S.Debugf("user synchronized with %q groups from provider %q", *newGroups, provider.Name)

	if err := data.AssignIdentityToGroups(db, user, provider, *newGroups); err != nil {
		return fmt.Errorf("assign identity to groups: %w", err)
	}

	return nil
}

// checkMemberOfGraphGroups calls the Microsoft Graph API to find out what groups a user belongs to
func checkMemberOfGraphGroups(accessToken string) (*[]string, error) {
	bearer := "Bearer " + accessToken

	req, err := http.NewRequest(http.MethodGet, graphGroupMemberEndpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create azure groups request: %w", err)
	}

	req.Header.Add("Authorization", bearer)

	ctx, cancel := context.WithTimeout(context.Background(), oidcProviderRequestTimeout)
	defer cancel()

	client := http.DefaultClient
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

	return &groups, nil
}
