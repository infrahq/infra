package api

// DestinationKubernetes struct for DestinationKubernetes
type DestinationKubernetes struct {
	CA       string `json:"ca" validate:"required"`
	Endpoint string `json:"endpoint" validate:"required"`
}
