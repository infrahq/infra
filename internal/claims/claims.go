package claims

type Custom struct {
	Email    string   `json:"email" validate:"required_without=Machine,excluded_with=Machine"`
	Machine  string   `json:"machine" validate:"required_without=Email,excluded_with=User,excluded_with=Group"`
	Groups   []string `json:"groups" validate:"excluded_with=Machine"`
	Nonce    string   `json:"nonce" validate:"required"`
	Provider string   `json:"provider" validate:"required_without=Machine,excluded_with=Machine"`
}
