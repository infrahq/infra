package claims

type Custom struct {
	Email  string   `json:"email" validate:"required"`
	Groups []string `json:"groups" validate:"required"`
	Nonce  string   `json:"nonce" validate:"required"`
}
