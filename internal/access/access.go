package access

import (
	"errors"
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/infrahq/infra/internal/server/data"
	"github.com/infrahq/infra/uid"
)

const ResourceInfraAPI = "infra"

// RequireInfraRole checks that the identity in the context can perform an action on a resource based on their granted roles
func RequireInfraRole(c *gin.Context, oneOfRoles ...string) (data.GormTxn, error) {
	rCtx := GetRequestContext(c)
	if err := IsAuthorized(rCtx, oneOfRoles...); err != nil {
		return nil, err
	}
	return rCtx.DBTxn, nil
}

var ErrNotAuthorized = errors.New("not authorized")

// AuthorizationError indicates that the user who performed the operation does
// not have the required role.
type AuthorizationError struct {
	Resource      string
	Operation     string
	RequiredRoles []string
}

func (e AuthorizationError) Error() string {
	var roles strings.Builder
	switch len(e.RequiredRoles) {
	case 1:
		roles.WriteString(e.RequiredRoles[0])
	default:
		for i, role := range e.RequiredRoles {
			roles.WriteString(role)
			switch {
			case i+1 == len(e.RequiredRoles)-1:
				roles.WriteString(", or ")
			case i+1 != len(e.RequiredRoles):
				roles.WriteString(", ")
			}
		}
	}
	return fmt.Sprintf("you do not have permission to %v %v, requires role %v",
		e.Operation, e.Resource, roles.String())
}

func (e AuthorizationError) Is(other error) bool {
	// nolint:errorlint // comparing with == is correct here, the caller uses Unwrap.
	return other == ErrNotAuthorized
}

func HandleAuthErr(err error, resource, operation string, roles ...string) error {
	if !errors.Is(err, ErrNotAuthorized) {
		return err
	}
	return AuthorizationError{
		Resource:      resource,
		Operation:     operation,
		RequiredRoles: roles,
	}
}

// IsAuthorized checks if the request has permission to perform the action. The
// request has permission if the user or one of the groups they belong to
// has a grant with one of the required roles.
// The resource is always ResourceInfraAPI.
func IsAuthorized(rCtx RequestContext, requiredRole ...string) error {
	user := rCtx.Authenticated.User
	if user == nil {
		return fmt.Errorf("no authenticated user")
	}
	grants, err := data.ListGrants(rCtx.DBTxn, data.ListGrantsOptions{
		Pagination:                 &data.Pagination{Limit: 1},
		BySubject:                  uid.NewIdentityPolymorphicID(user.ID),
		ByPrivileges:               requiredRole,
		ByResource:                 ResourceInfraAPI,
		IncludeInheritedFromGroups: true,
	})
	if err != nil {
		return fmt.Errorf("has grants: %w", err)
	}
	if len(grants) == 0 {
		return ErrNotAuthorized
	}
	return nil
}
