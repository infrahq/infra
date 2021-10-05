package cmd

type ClientConfigV0dot1 struct {
	Version       string `json:"version"` // always blank
	Name          string `json:"name"`
	Host          string `json:"host"`
	Token         string `json:"token"`
	SkipTLSVerify bool   `json:"skip-tls-verify"`
	SourceID      string `json:"source-id"`
}

// ToV1dot0 upgrades the config to the 1.0 verison
func (c ClientConfigV0dot1) ToV1dot0() *ClientConfigV1dot0 {
	return &ClientConfigV1dot0{
		Version: "1.0",
		Registries: []RegistryConfig{
			{
				Name:          c.Name,
				Host:          c.Host,
				Token:         c.Token,
				SkipTLSVerify: c.SkipTLSVerify,
				SourceID:      c.SourceID,
				Current:       true,
			},
		},
	}
}
