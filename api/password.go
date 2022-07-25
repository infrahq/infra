package api

type PasswordResponse struct {
	LowercaseMin int `json:"lowercaseMin"`
	UppercaseMin int `json:"uppercaseMin"`
	NumberMin    int `json:"numberMin"`
	SymbolMin    int `json:"symbolMin"`
	LengthMin    int `json:"lengthMin"`
}
