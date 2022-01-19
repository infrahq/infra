package api

// GrantKubernetes struct for GrantKubernetes
type GrantKubernetes struct {
	Kind      GrantKubernetesKind `json:"kind"`
	Name      string              `json:"name"`
	Namespace string              `json:"namespace"`
}

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
