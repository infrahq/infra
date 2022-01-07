package api

// GrantKubernetes struct for GrantKubernetes
type GrantKubernetes struct {
	Kind      GrantKubernetesKind `json:"kind"`
	Name      string              `json:"name"`
	Namespace string              `json:"namespace"`
}
