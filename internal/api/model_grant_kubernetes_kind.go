package api

// GrantKubernetesKind the model 'GrantKubernetesKind'
type GrantKubernetesKind string

// List of GrantKubernetesKind
const (
	GrantKubernetesKindRole        GrantKubernetesKind = "role"
	GrantKubernetesKindClusterRole GrantKubernetesKind = "cluster-role"
)

var ValidGrantKubernetesKinds = []GrantKubernetesKind{
	GrantKubernetesKindRole,
	GrantKubernetesKindClusterRole,
}
