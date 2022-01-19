package api

// LoginRequest struct for LoginRequest
type LoginRequest struct {
	Okta *LoginRequestOkta `json:"okta,omitempty"`
}

// LoginResponse struct for LoginResponse
type LoginResponse struct {
	Token string `json:"token"`
	Name  string `json:"name"`
}

// LoginRequestOkta struct for LoginRequestOkta
type LoginRequestOkta struct {
	Domain string `json:"domain" validate:"fqdn,required"`
	Code   string `json:"code" validate:"required"`
}
