package api

// LoginRequest struct for LoginRequest
type LoginRequest struct {
	Okta *LoginRequestOkta `json:"okta,omitempty"`
}
