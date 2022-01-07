package api

// GrantKubernetesKind the model 'GrantKubernetesKind'
type GrantKubernetesKind string

// List of GrantKubernetesKind
const (
	GRANTKUBERNETESKIND_ROLE         GrantKubernetesKind = "role"
	GRANTKUBERNETESKIND_CLUSTER_ROLE GrantKubernetesKind = "cluster-role"
)

var allowedGrantKubernetesKindEnumValues = []GrantKubernetesKind{
	"role",
	"cluster-role",
}
