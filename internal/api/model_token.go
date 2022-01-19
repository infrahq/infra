package api

// Token struct for Token
type Token struct {
	Token   string `json:"token"`
	Expires int64  `json:"expires"`
}

// TokenRequest struct for TokenRequest
type TokenRequest struct {
	Destination string `json:"destination" validate:"required"`
}
