package server

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/internal/access"
	"github.com/infrahq/infra/internal/server/models"
	"github.com/infrahq/infra/internal/server/providers"
)

// caution: this endpoint is unauthenticated, do not return sensitive info
func (a *API) ListProviders(c *gin.Context, r *api.ListProvidersRequest) (*api.ListResponse[api.Provider], error) {
	exclude := []models.ProviderKind{models.ProviderKindInfra}
	p := models.RequestToPagination(r.PaginationRequest)
	providers, err := access.ListProviders(c, r.Name, exclude, &p)
	if err != nil {
		return nil, err
	}

	result := api.NewListResponse(providers, models.PaginationToResponse(p), func(provider models.Provider) api.Provider {
		return *provider.ToAPI()
	})

	return result, nil
}

// caution: this endpoint is unauthenticated, do not return sensitive info
func (a *API) GetProvider(c *gin.Context, r *api.Resource) (*api.Provider, error) {
	provider, err := access.GetProvider(c, r.ID)
	if err != nil {
		return nil, err
	}

	return provider.ToAPI(), nil
}

var (
	dashAdminRemover = regexp.MustCompile(`(.*)\-admin(\.okta\.com)`)
	protocolRemover  = regexp.MustCompile(`http[s]?://`)
)

func cleanupURL(url string) string {
	url = strings.TrimSpace(url)
	url = dashAdminRemover.ReplaceAllString(url, "$1$2")
	url = protocolRemover.ReplaceAllString(url, "")

	return url
}

func (a *API) CreateProvider(c *gin.Context, r *api.CreateProviderRequest) (*api.Provider, error) {
	provider := &models.Provider{
		Name:         r.Name,
		URL:          cleanupURL(r.URL),
		ClientID:     r.ClientID,
		ClientSecret: models.EncryptedAtRest(r.ClientSecret),
	}

	if r.API != nil {
		// the private key PEM needs to have its newline formatted, the API does not allow new-line formatting inputs
		provider.PrivateKey = models.EncryptedAtRest(strings.ReplaceAll(string(r.API.PrivateKey), "\\n", "\n"))
		provider.ClientEmail = r.API.ClientEmail
		provider.DomainAdminEmail = r.API.DomainAdminEmail
	}

	kind, err := models.ParseProviderKind(r.Kind)
	if err != nil {
		return nil, err
	}
	provider.Kind = kind

	if err := a.setProviderInfoFromServer(c, provider); err != nil {
		return nil, err
	}

	if err := access.CreateProvider(c, provider); err != nil {
		return nil, err
	}

	return provider.ToAPI(), nil
}

func (a *API) UpdateProvider(c *gin.Context, r *api.UpdateProviderRequest) (*api.Provider, error) {
	provider := &models.Provider{
		Model: models.Model{
			ID: r.ID,
		},
		Name:         r.Name,
		URL:          cleanupURL(r.URL),
		ClientID:     r.ClientID,
		ClientSecret: models.EncryptedAtRest(r.ClientSecret),
	}

	if r.API != nil {
		// the private key PEM needs to have its newline formatted, the API does not allow new-line formatting inputs
		provider.PrivateKey = models.EncryptedAtRest(strings.ReplaceAll(string(r.API.PrivateKey), "\\n", "\n"))
		provider.ClientEmail = r.API.ClientEmail
		provider.DomainAdminEmail = r.API.DomainAdminEmail
	}

	kind, err := models.ParseProviderKind(r.Kind)
	if err != nil {
		return nil, err
	}
	provider.Kind = kind

	if err := a.setProviderInfoFromServer(c, provider); err != nil {
		return nil, err
	}

	if err := access.SaveProvider(c, provider); err != nil {
		return nil, err
	}

	return provider.ToAPI(), nil
}

func (a *API) DeleteProvider(c *gin.Context, r *api.Resource) (*api.EmptyResponse, error) {
	return nil, access.DeleteProvider(c, r.ID)
}

// setProviderInfoFromServer checks information provided by an OIDC server
func (a *API) setProviderInfoFromServer(c *gin.Context, provider *models.Provider) error {
	rCtx := getRequestContext(c)
	// create a provider client to validate the server and get its info
	oidc, err := newProviderOIDCClient(rCtx, provider, "http://localhost:8301")
	if err != nil {
		return fmt.Errorf("%w: %s", internal.ErrBadRequest, err)
	}

	err = oidc.Validate(c)
	if err != nil {
		if errors.Is(err, providers.ErrValidation) {
			return fmt.Errorf("%w: %s", internal.ErrBadRequest, err)
		}
		return err
	}

	authServerInfo, err := oidc.AuthServerInfo(c)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return fmt.Errorf("%w: %s", internal.ErrBadGateway, err)
		}
		return err
	}

	provider.AuthURL = authServerInfo.AuthURL
	provider.Scopes = authServerInfo.ScopesSupported

	return nil
}
