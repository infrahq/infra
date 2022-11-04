package server

import (
	"github.com/infrahq/infra/internal/access"
)

type authorization struct {
	OneOfRoles []string
	Resource   string
	Operation  string
	// AuthorizedByID is a custom function that can be set by a route to
	// authorize the request. Generally the function should compare the
	// requested ID to the rCtx.Authenticated.User.ID to see if the request is
	// for the authorized user.
	// The request will be authorized if AuthorizedByID returns nil. Any error
	// instructs IsAuthorized to proceed to authorizing by role.
	AuthorizeByID func(rCtx access.RequestContext, req any) error
}

func (a authorization) IsAuthorized(rCtx access.RequestContext, req any) error {
	if a.AuthorizeByID != nil {
		if err := a.AuthorizeByID(rCtx, req); err == nil {
			return nil
		}
	}

	err := access.IsAuthorized(rCtx, a.OneOfRoles...)
	return access.HandleAuthErr(err, a.Resource, a.Operation, a.OneOfRoles...)
}

func requireRole(resource, operation string, oneOfRole ...string) *authorization {
	return &authorization{
		OneOfRoles: oneOfRole,
		Resource:   resource,
		Operation:  operation,
	}
}

type noAuthorization struct{}

func (n noAuthorization) IsAuthorized(access.RequestContext, any) error {
	return nil
}
