package server

import (
	"fmt"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal/access"
	"github.com/infrahq/infra/internal/server/models"
)

func (a *API) ListOrganizations(rCtx access.RequestContext, r *api.ListOrganizationsRequest) (*api.ListResponse[api.Organization], error) {
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

func (a *API) GetOrganization(rCtx access.RequestContext, r *api.GetOrganizationRequest) (*api.Organization, error) {
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

func (a *API) CreateOrganization(rCtx access.RequestContext, r *api.CreateOrganizationRequest) (*api.Organization, error) {
	org := &models.Organization{
		Name:      r.Name,
		Domain:    r.Domain,
		InstallID: rCtx.DataDB.DefaultOrg.InstallID,
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

func (a *API) DeleteOrganization(rCtx access.RequestContext, r *api.Resource) (*api.EmptyResponse, error) {
	return nil, access.DeleteOrganization(rCtx, r.ID)
}

func (a *API) UpdateOrganization(rCtx access.RequestContext, r *api.UpdateOrganizationRequest) (*api.Organization, error) {
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
