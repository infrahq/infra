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
	"github.com/infrahq/infra/internal/generate"
	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/internal/server/models"
)

// caution: this endpoint is unauthenticated, do not return sensitive info
func (a *API) ListProviders(c *gin.Context, r *api.ListProvidersRequest) (*api.ListResponse[api.Provider], error) {
	rCtx := getRequestContext(c)
	p := PaginationFromRequest(r.PaginationRequest)
	opts := data.ListProvidersOptions{
		ByName:               r.Name,
		ExcludeInfraProvider: true,
		Pagination:           &p,
	}
	providers, err := data.ListProviders(rCtx.DBTxn, opts)
	if err != nil {
		return nil, err
	}

	// if social login is configured, also return that option
	if a.server.Google != nil {
		providers = append(providers, *a.server.Google)
	}

	result := api.NewListResponse(providers, PaginationToResponse(p), func(provider models.Provider) api.Provider {
		return *provider.ToAPI()
	})

	return result, nil
}

// caution: this endpoint is unauthenticated, do not return sensitive info
func (a *API) GetProvider(c *gin.Context, r *api.Resource) (*api.Provider, error) {
	rCtx := getRequestContext(c)
	provider, err := data.GetProvider(rCtx.DBTxn, data.GetProviderOptions{ByID: r.ID})
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
	rCtx := getRequestContext(c)
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

	// If name is not provided, generate based on provider kind
	if provider.Name == "" {
		provider.Name = provider.Kind.String()

		// If provider name is taken, generate a random tag
		providers, err := data.ListProviders(rCtx.DBTxn, data.ListProvidersOptions{
			ByName: provider.Kind.String(),
		})
		if err != nil {
			return nil, fmt.Errorf("Error while generating name for provider: %w", err)
		}
		if len(providers) > 0 {
			randomString, err := generate.CryptoRandom(6, generate.CharsetAlphaNumericNoVowels)
			if err != nil {
				return nil, fmt.Errorf("Error while generating name for provider: %w", err)
			}

			provider.Name = r.Kind + "-" + randomString
		}
	}

	if err := a.setProviderInfoFromServer(rCtx.Request.Context(), provider); err != nil {
		return nil, err
	}

	if err := access.CreateProvider(c, provider); err != nil {
		return nil, err
	}

	return provider.ToAPI(), nil
}

func (a *API) PatchProvider(c *gin.Context, r *api.PatchProviderRequest) (*api.Provider, error) {
	rCtx := getRequestContext(c)
	provider, err := data.GetProvider(rCtx.DBTxn, data.GetProviderOptions{ByID: r.ID})
	if err != nil {
		return nil, err
	}
	if r.Name != "" {
		provider.Name = r.Name
	}
	if r.ClientSecret != "" {
		provider.ClientSecret = models.EncryptedAtRest(r.ClientSecret)
	}
	if err = access.SaveProvider(c, provider); err != nil {
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

	if err := a.setProviderInfoFromServer(c.Request.Context(), provider); err != nil {
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
func (a *API) setProviderInfoFromServer(ctx context.Context, provider *models.Provider) error {
	// create a provider client to validate the server and get its info
	oidc, err := a.server.providerClient(ctx, provider, "")
	if err != nil {
		return fmt.Errorf("%w: %s", internal.ErrBadRequest, err)
	}

	err = oidc.Validate(ctx)
	if err != nil {
		return err
	}

	authServerInfo, err := oidc.AuthServerInfo(ctx)
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
