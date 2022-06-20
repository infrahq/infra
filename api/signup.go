package api

type SignupEnabledResponse struct {
	Enabled bool `json:"enabled"`
}

type SignupRequest struct {
	Name     string `json:"name" validate:"omitempty,email,required_without=Email"`
	Email    string `json:"email" validate:"omitempty,email,required_without=Name"` // #1825: remove, this is for migration
	Password string `json:"password" validate:"required"`
}
