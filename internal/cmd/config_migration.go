package cmd

import "github.com/infrahq/infra/uid"

type ClientConfigV0dot1 struct {
	Version       string `json:"version"` // always blank in v0.1
	Name          string `json:"name"`
	Host          string `json:"host"`
	Token         string `json:"token"`
	SkipTLSVerify bool   `json:"skip-tls-verify"`
	SourceID      string `json:"source-id"`
}

type ClientConfigV0dot2 struct {
	Version string                   `json:"version"` // v0.2
	Hosts   []ClientHostConfigV0dot2 `json:"hosts"`
}

type ClientHostConfigV0dot2 struct {
	Name          string `json:"name"`
	Host          string `json:"host"`
	Token         string `json:"token"`
	SkipTLSVerify bool   `json:"skip-tls-verify"`
	SourceID      string `json:"source-id"`
	Current       bool   `json:"current"`
}

// ToV0dot2 upgrades the config to the 0.2 version
func (c ClientConfigV0dot1) ToV0dot2() *ClientConfigV0dot2 {
	return &ClientConfigV0dot2{
		Version: "0.2",
		Hosts: []ClientHostConfigV0dot2{
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

// ToV0dot3 upgrades the config to the 0.3 version
func (c ClientConfigV0dot2) ToV0dot3() *ClientConfig {
	conf := &ClientConfig{
		Version: "0.3",
	}

	for _, h := range c.Hosts {
		providerID := uid.New()
		providerID.UnmarshalText([]byte(h.SourceID))

		conf.Hosts = append(conf.Hosts, ClientHostConfig{
			Name:          h.Name,
			Host:          h.Host,
			Token:         h.Token,
			SkipTLSVerify: h.SkipTLSVerify,
			ProviderID:    providerID,
			Current:       h.Current,
		})
	}

	return conf
}
