package api

type SignupEnabledResponse struct {
	Enabled bool `json:"enabled"`
}

type SignupRequest struct {
	Name     string `json:"name" validate:"required"`
	Password string `json:"password" validate:"required"`
}
