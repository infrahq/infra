package access

import "fmt"

type Role string

const (
	RoleSupportAdmin = "support-admin"
	RoleAdmin        = "admin"
	RoleView         = "view"
	RoleConnector    = "connector"
)

type Resource int

const (
	ResourceSystem        Resource = 0
	ResourceGrants        Resource = 1
	ResourceProviders     Resource = 2
	ResourceOrganizations Resource = 3
	ResourceDestinations  Resource = 4
	ResourceUsers         Resource = 5
	ResourceGroups        Resource = 6
	ResourceAccessKeys    Resource = 7
	ResourceCredentials   Resource = 8
	ResourceSettings      Resource = 9
)

func (r Resource) String() string {
	switch r {
	case ResourceSystem:
		return "system"
	case ResourceGrants:
		return "grants"
	case ResourceProviders:
		return "providers"
	case ResourceOrganizations:
		return "organizations"
	case ResourceDestinations:
		return "destinations"
	case ResourceUsers:
		return "users"
	case ResourceGroups:
		return "groups"
	case ResourceAccessKeys:
		return "access keys"
	case ResourceCredentials:
		return "credentials"
	default:
		return ""
	}
}

type Operation int

const (
	OperationRead  Operation = 0
	OperationWrite Operation = 1
)

func (p Operation) String() string {
	switch p {
	case OperationRead:
		return "read"
	case OperationWrite:
		return "write"
	default:
		return ""
	}
}

type Access struct {
	Resource  Resource
	Operation Operation
}

func perm(r Resource, o Operation) Access {
	return Access{Resource: r, Operation: o}
}

type PermissionSet map[Access]struct{}

func toSet(access ...Access) PermissionSet {
	result := make(PermissionSet, len(access))
	for _, a := range access {
		result[a] = struct{}{}
	}
	return result
}

func (p PermissionSet) Allows(access Access) error {
	if _, ok := p[access]; !ok {
		return fmt.Errorf("missing permission to %v the %v resource",
			access.Operation, access.Resource)
	}
	return nil
}

var RolePermissions = map[string]PermissionSet{
	RoleView: toSet(
		perm(ResourceAccessKeys, OperationRead),
		perm(ResourceGrants, OperationRead),
		perm(ResourceGroups, OperationRead),
		perm(ResourceUsers, OperationRead),
	),

	RoleAdmin: toSet(
		perm(ResourceAccessKeys, OperationRead),
		perm(ResourceAccessKeys, OperationWrite),
		perm(ResourceCredentials, OperationWrite),
		perm(ResourceDestinations, OperationRead),
		perm(ResourceDestinations, OperationWrite),
		perm(ResourceGrants, OperationRead),
		perm(ResourceGrants, OperationWrite),
		perm(ResourceGroups, OperationRead),
		perm(ResourceGroups, OperationWrite),
		perm(ResourceUsers, OperationRead),
		perm(ResourceUsers, OperationWrite),
		perm(ResourceProviders, OperationRead),
		perm(ResourceProviders, OperationWrite),
		perm(ResourceSettings, OperationWrite),
	),

	RoleConnector: toSet(
		perm(ResourceDestinations, OperationRead),
		perm(ResourceDestinations, OperationWrite),
		perm(ResourceGrants, OperationRead),
		perm(ResourceUsers, OperationRead),
		perm(ResourceGroups, OperationRead),
	),

	RoleSupportAdmin: toSet(
		perm(ResourceOrganizations, OperationRead),
		perm(ResourceOrganizations, OperationWrite),
		perm(ResourceSystem, OperationRead),
	),
}
