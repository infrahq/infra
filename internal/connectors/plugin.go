package connectors

import (
	"context"

	"github.com/infrahq/infra/internal/server/models"
)

type pluginInitFunc func(config interface{}) (AuthorizerPlugin, error)

var plugins = map[string]pluginInitFunc{}

func Register(name string, initFn pluginInitFunc) {
	plugins[name] = initFn
}

type GrantRecord struct {
	Grant     models.Grant
	User      *models.Identity
	GroupData *GroupRecord
	Role      *RoleRecord
}

type RoleRecord struct {
	Name string
}

type GroupRecord struct {
	Group models.Group
	Users []models.Identity
}

type AuthorizerPlugin interface {
	Run(context.Context) error

	// AddAccess(g GrantRecord) error
	// RemoveAccess(g GrantRecord) error

	// problem with syncGrants is that it might be very difficult to pull the state of access out of the destination system.
	// in k8s we tag everything so we know what's what, but that might not
	SyncGrants(grants []GrantRecord) error
}

// type AuthenticatorPlugin interface {
// 	AddCredential(c *credential) error
// 	RemoveCredential(c *credential) error
// }
