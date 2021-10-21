package cmd

type ClientConfigV0dot1 struct {
	Version       string `json:"version"` // always blank in v0.1
	Name          string `json:"name"`
	Host          string `json:"host"`
	Token         string `json:"token"`
	SkipTLSVerify bool   `json:"skip-tls-verify"`
	SourceID      string `json:"source-id"`
}

// ToV0dot2 upgrades the config to the 0.2 version
func (c ClientConfigV0dot1) ToV0dot2() *ClientConfigV0dot2 {
	return &ClientConfigV0dot2{
		Version: "0.2",
		Registries: []ClientRegistryConfig{
			{
				Name:          c.Name,
				Host:          c.Host,
				Token:         c.Token,
				SkipTLSVerify: c.SkipTLSVerify,
				Current:       true,
			},
		},
	}
}
