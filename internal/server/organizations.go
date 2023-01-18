package server

import (
	"fmt"

	"github.com/gin-gonic/gin"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal/access"
	"github.com/infrahq/infra/internal/server/models"
)

func (a *API) ListOrganizations(c *gin.Context, r *api.ListOrganizationsRequest) (*api.ListResponse[api.Organization], error) {
	rCtx := getRequestContext(c)
	p := PaginationFromRequest(r.PaginationRequest)
	orgs, err := access.ListOrganizations(rCtx, r.Name, &p)
	if err != nil {
		return nil, err
	}

	result := api.NewListResponse(orgs, PaginationToResponse(p), func(org models.Organization) api.Organization {
		return *org.ToAPI()
	})

	return result, nil
}

func (a *API) GetOrganization(c *gin.Context, r *api.GetOrganizationRequest) (*api.Organization, error) {
	rCtx := getRequestContext(c)
	if r.ID.IsSelf {
		iden := rCtx.Authenticated.Organization
		if iden == nil {
			return nil, fmt.Errorf("no authenticated user")
		}
		r.ID.ID = iden.ID
	}
	org, err := access.GetOrganization(rCtx, r.ID.ID)
	if err != nil {
		return nil, err
	}

	return org.ToAPI(), nil
}

func (a *API) CreateOrganization(c *gin.Context, r *api.CreateOrganizationRequest) (*api.Organization, error) {
	rCtx := getRequestContext(c)
	org := &models.Organization{
		Name:   r.Name,
		Domain: r.Domain,
	}

	authIdent := rCtx.Authenticated.User
	if authIdent != nil {
		org.CreatedBy = authIdent.ID
	}

	err := access.CreateOrganization(rCtx, org)
	if err != nil {
		return nil, err
	}

	a.t.Org(org.ID.String(), authIdent.ID.String(), org.Name, org.Domain)

	return org.ToAPI(), nil
}

func (a *API) DeleteOrganization(c *gin.Context, r *api.Resource) (*api.EmptyResponse, error) {
	return nil, access.DeleteOrganization(getRequestContext(c), r.ID)
}

func (a *API) UpdateOrganization(c *gin.Context, r *api.UpdateOrganizationRequest) (*api.Organization, error) {
	rCtx := getRequestContext(c)
	org, err := access.GetOrganization(rCtx, r.ID)
	if err != nil {
		return nil, err
	}

	// overwrite the existing domains to the incoming ones
	org.AllowedDomains = []string{}
	// remove duplicate domains
	domains := make(map[string]bool)
	for _, d := range r.AllowedDomains {
		if !domains[d] {
			org.AllowedDomains = append(org.AllowedDomains, d)
		}
		domains[d] = true
	}

	err = access.UpdateOrganization(rCtx, org)
	if err != nil {
		return nil, err
	}

	return org.ToAPI(), nil
}
