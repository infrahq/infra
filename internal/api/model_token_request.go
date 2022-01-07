package api

// TokenRequest struct for TokenRequest
type TokenRequest struct {
	Destination string `json:"destination" validate:"required"`
}
