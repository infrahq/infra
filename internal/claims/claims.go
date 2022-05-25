package claims

type Custom struct {
	Name   string   `json:"name" validate:"required"`
	Groups []string `json:"groups"`
}
