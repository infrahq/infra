package claims

type Custom struct {
	Name   string   `json:"name"`
	Groups []string `json:"groups"`
	Nonce  string   `json:"nonce"`
}
