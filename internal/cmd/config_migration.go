package cmd

import (
	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/uid"
)

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
	SourceID      uid.ID `json:"source-id"`
	Current       bool   `json:"current"`
}

type ClientConfigV0dot3 struct {
	Version string                   `json:"version"`
	Hosts   []ClientHostConfigV0dot3 `json:"hosts"`
}

type ClientHostConfigV0dot3 struct {
	PolymorphicID uid.PolymorphicID `json:"polymorphic-id"`
	Name          string            `json:"name"` // user name
	Host          string            `json:"host"`
	AccessKey     string            `json:"access-key,omitempty"`
	SkipTLSVerify bool              `json:"skip-tls-verify"` // where is the other cert info stored?
	ProviderID    uid.ID            `json:"provider-id,omitempty"`
	Expires       api.Time          `json:"expires"`
	Current       bool              `json:"current"`
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
func (c ClientConfigV0dot2) ToV0dot3() *ClientConfigV0dot3 {
	conf := &ClientConfigV0dot3{
		Version: "0.3",
	}

	for _, h := range c.Hosts {
		providerID := uid.New()
		h.SourceID = providerID

		conf.Hosts = append(conf.Hosts, ClientHostConfigV0dot3{
			Name:          h.Name,
			Host:          h.Host,
			AccessKey:     h.Token,
			SkipTLSVerify: h.SkipTLSVerify,
			ProviderID:    providerID,
			Current:       h.Current,
		})
	}

	return conf
}

// ToV0dot4 upgrades the config to the 0.4 version
func (c ClientConfigV0dot3) ToV0dot4() *ClientConfig {
	conf := &ClientConfig{
		ClientConfigVersion: ClientConfigVersion{
			Version: "0.4",
		},
	}

	for _, h := range c.Hosts {
		userID, _ := h.PolymorphicID.ID()
		conf.Hosts = append(conf.Hosts, ClientHostConfig{
			UserID:        userID,
			Name:          h.Name,
			Host:          h.Host,
			AccessKey:     h.AccessKey,
			SkipTLSVerify: h.SkipTLSVerify,
			ProviderID:    h.ProviderID,
			Expires:       h.Expires,
			Current:       h.Current,
		})
	}

	return conf
}
