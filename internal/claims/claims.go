package claims

type Custom struct {
	Name   string   `json:"name"`
	Groups []string `json:"groups"`
}
