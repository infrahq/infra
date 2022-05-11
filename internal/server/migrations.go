package server

import (
	"net/http"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/uid"
)

func (a *API) addRequestRewrites() {
	// all request migrations go here
	//
	//    http method ---v           v--- path      v--- last version that supports the old response
	type deprecatedListAccessKeysRequest struct {
		IdentityID uid.ID `form:"identity_id"`
		Name       string `form:"name"`
	}
	addRequestRewrite(a, http.MethodGet, "/access-keys", "0.12.2", func(old deprecatedListAccessKeysRequest) api.ListAccessKeysRequest {
		return api.ListAccessKeysRequest{
			UserID: old.IdentityID,
			Name:   old.Name,
		}
	})
	type deprecatedCreateAccessKeyRequest struct {
		IdentityID        uid.ID       `json:"identityID" validate:"required"`
		Name              string       `json:"name" validate:"excludes= "`
		TTL               api.Duration `json:"ttl" validate:"required"`
		ExtensionDeadline api.Duration `json:"extensionDeadline,omitempty" validate:"required"`
	}
	addRequestRewrite(a, http.MethodPost, "/access-keys", "0.12.2", func(old deprecatedCreateAccessKeyRequest) api.CreateAccessKeyRequest {
		return api.CreateAccessKeyRequest{
			UserID:            old.IdentityID,
			Name:              old.Name,
			TTL:               old.TTL,
			ExtensionDeadline: old.ExtensionDeadline,
		}
	})
	type deprecatedListGrantsRequest struct {
		Identity  uid.ID `form:"identity" validate:"excluded_with=Group"`
		Group     uid.ID `form:"group" validate:"excluded_with=Identity"`
		Resource  string `form:"resource"`
		Privilege string `form:"privilege"`
	}
	addRequestRewrite(a, http.MethodGet, "/v1/grants", "0.12.2", func(old deprecatedListGrantsRequest) api.ListGrantsRequest {
		return api.ListGrantsRequest{
			User:      old.Identity,
			Group:     old.Group,
			Privilege: old.Privilege,
			Resource:  old.Resource,
		}
	})
	type deprecatedCreateGrantRequest struct {
		Identity  uid.ID `json:"identity" validate:"required_without=Group"`
		Group     uid.ID `json:"group" validate:"required_without=Identity"`
		Privilege string `json:"privilege" validate:"required"`
		Resource  string `json:"resource" validate:"required"`
	}
	addRequestRewrite(a, http.MethodPost, "/v1/grants", "0.12.2", func(old deprecatedCreateGrantRequest) api.CreateGrantRequest {
		return api.CreateGrantRequest{
			User:      old.Identity,
			Group:     old.Group,
			Privilege: old.Privilege,
			Resource:  old.Resource,
		}
	})
	// next migration ...
}

func (a *API) addResponseRewrites() {
	// all response migrations go here
	//
	//    http method ---v           v--- path      v--- last version that supports the old response
	addResponseRewrite(a, "get", "/v1/access-keys", "0.12.2", func(newResponse *api.ListResponse[api.AccessKey]) []api.AccessKey {
		return newResponse.Items
	})
	addResponseRewrite(a, "get", "/v1/identities", "0.12.2", func(newResponse *api.ListResponse[api.User]) []api.User {
		return newResponse.Items
	})
	addResponseRewrite(a, "get", "/v1/identities/:id/grants", "0.12.2", func(newResponse *api.ListResponse[api.Grant]) []api.Grant {
		return newResponse.Items
	})
	addResponseRewrite(a, "get", "/v1/identities/:id/groups", "0.12.2", func(newResponse *api.ListResponse[api.Group]) []api.Group {
		return newResponse.Items
	})
	addResponseRewrite(a, "get", "/v1/groups", "0.12.2", func(newResponse *api.ListResponse[api.Group]) []api.Group {
		return newResponse.Items
	})
	addResponseRewrite(a, "get", "/v1/groups/:id/grants", "0.12.2", func(newResponse *api.ListResponse[api.Grant]) []api.Grant {
		return newResponse.Items
	})
	addResponseRewrite(a, "get", "/v1/providers", "0.12.2", func(newResponse *api.ListResponse[api.Provider]) []api.Provider {
		return newResponse.Items
	})
	addResponseRewrite(a, "get", "/v1/destinations", "0.12.2", func(newResponse *api.ListResponse[api.Destination]) []api.Destination {
		return newResponse.Items
	})
	addResponseRewrite(a, http.MethodGet, "/v1/grants", "0.12.2", func(newResponse *api.ListResponse[api.Grant]) []identityGrant {
		resp := []identityGrant{}

		for _, item := range newResponse.Items {
			resp = append(resp, migrateUserGrantToIdentity(item))
		}

		return resp
	})
	addResponseRewrite(a, http.MethodPost, "/v1/grants", "0.12.2", func(newResponse *api.Grant) identityGrant {
		return migrateUserGrantToIdentity(*newResponse)
	})
	addResponseRewrite(a, http.MethodGet, "/v1/grants/:id", "0.12.2", func(newResponse *api.Grant) identityGrant {
		return migrateUserGrantToIdentity(*newResponse)
	})
	// next migration...
}

func (a *API) addRewrites() {
	a.addRequestRewrites()
	a.addResponseRewrites()
}

// addRedirects for API endpoints that have moved to a different path
func (a *API) addRedirects() {
	addRedirect(a, http.MethodGet, "/v1/identities", "/v1/users", "0.12.2")
	addRedirect(a, http.MethodPost, "/v1/identities", "/v1/users", "0.12.2")
	addRedirect(a, http.MethodGet, "/v1/identities/:id", "/v1/users/:id", "0.12.2")
	addRedirect(a, http.MethodPut, "/v1/identities/:id", "/v1/users/:id", "0.12.2")
	addRedirect(a, http.MethodDelete, "/v1/identities/:id", "/v1/users/:id", "0.12.2")

	addRedirect(a, http.MethodGet, "/v1/identities/:id/groups", "/v1/users/:id/groups", "0.12.2")
	addRedirect(a, http.MethodGet, "/v1/identities/:id/grants", "/v1/users/:id/grants", "0.12.2")
}

type identityGrant struct {
	ID uid.ID `json:"id"`

	Created   api.Time `json:"created"`
	CreatedBy uid.ID   `json:"created_by"`
	Updated   api.Time `json:"updated"`

	Subject   uid.PolymorphicID `json:"subject,omitempty"`
	Privilege string            `json:"privilege"`
	Resource  string            `json:"resource"`
}

func migrateUserGrantToIdentity(grant api.Grant) identityGrant {
	var sub uid.PolymorphicID

	if grant.User != 0 {
		sub = uid.NewIdentityPolymorphicID(grant.User)
	} else {
		sub = uid.NewGroupPolymorphicID(grant.Group)
	}

	return identityGrant{
		ID:        grant.ID,
		Created:   grant.Created,
		CreatedBy: grant.CreatedBy,
		Updated:   grant.Updated,
		Subject:   sub,
		Group:     grant.Group,
		Privilege: grant.Privilege,
		Resource:  grant.Resource,
	}
}
