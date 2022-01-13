package api

// Token struct for Token
type Token struct {
	Token   string `json:"token"`
	Expires int64  `json:"expires"`
}
