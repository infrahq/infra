package api

type CreateTokenResponse struct {
	Expires Time   `json:"expires"`
	Token   string `json:"token"`
}
