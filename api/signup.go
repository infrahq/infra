package api

type SignupEnabledResponse struct {
	Enabled bool `json:"enabled"`
}

type SignupRequest struct {
	Name     string `json:"name"`
	Password string `json:"password"`
}
