package cmd

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
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
	Version string                   `json:"version"` // always 0.2 in v0.2
	Hosts   []ClientHostConfigV0dot2 `json:"hosts"`
}

// current: v0.3
type ClientConfig struct {
	Version string             `json:"version"`
	Hosts   []ClientHostConfig `json:"hosts"`
}

type ClientHostConfigV0dot2 struct {
	Name          string `json:"name"`
	Host          string `json:"host"`
	Token         string `json:"token"`
	SkipTLSVerify bool   `json:"skip-tls-verify"`
	SourceID      string `json:"source-id"`
	Current       bool   `json:"current"`
}

// when you change this, match the config version to the bump it was changed in, otherwise leave it
// current: v0.3
type ClientHostConfig struct {
	Name          string `json:"name"`
	Host          string `json:"host"`
	Token         string `json:"token"`
	SkipTLSVerify bool   `json:"skip-tls-verify"` // where is the other cert info stored?
	ProviderID    string `json:"provider-id"`
	Current       bool   `json:"current"`
}

var ErrConfigNotFound = errors.New("could not read local credentials. Are you logged in? Use \"infra login\" to login")

func NewClientConfig() *ClientConfig {
	return &ClientConfig{
		Version: "0.3",
	}
}

func readConfig() (*ClientConfig, error) {
	config := &ClientConfig{}

	infraDir, err := infraHomeDir()
	if err != nil {
		return nil, err
	}

	contents, err := ioutil.ReadFile(filepath.Join(infraDir, "config"))
	if os.IsNotExist(err) {
		return nil, ErrConfigNotFound
	}

	if err != nil {
		return nil, err
	}

	if err = json.Unmarshal(contents, &config); err != nil {
		return nil, err
	}

	if config.Version == "" {
		// if version is missing, it needs to be updated to the latest
		configv0dot1 := ClientConfigV0dot1{}
		if err = json.Unmarshal(contents, &configv0dot1); err != nil {
			return nil, err
		}

		return configv0dot1.ToV0dot2().ToV0dot3(), nil
	} else if config.Version == "0.2" {
		configv0dot2 := ClientConfigV0dot2{}
		if err = json.Unmarshal(contents, &configv0dot2); err != nil {
			return nil, err
		}

		return configv0dot2.ToV0dot3(), nil
	}

	return config, nil
}

func writeConfig(config *ClientConfig) error {
	infraDir, err := infraHomeDir()
	if err != nil {
		return err
	}

	contents, err := json.Marshal(config)
	if err != nil {
		return err
	}

	if err = ioutil.WriteFile(filepath.Join(infraDir, "config"), []byte(contents), 0o600); err != nil {
		return err
	}

	return nil
}

func currentHostConfig() (*ClientHostConfig, error) {
	return readHostConfig("")
}

func readHostConfig(host string) (*ClientHostConfig, error) {
	cfg, err := readConfig()
	if err != nil {
		return nil, err
	}

	for i, c := range cfg.Hosts {
		if len(host) == 0 && c.Current {
			return &cfg.Hosts[i], nil
		}

		if c.Host == host {
			return &cfg.Hosts[i], nil
		}
	}

	return nil, nil
}

func removeHostConfig(host string) error {
	cfg, err := readConfig()
	if err != nil {
		return err
	}

	for i, c := range cfg.Hosts {
		if c.Host == host {
			cfg.Hosts = append(cfg.Hosts[:i], cfg.Hosts[i+1:]...)
			break
		}
	}

	err = writeConfig(cfg)
	if err != nil {
		return err
	}

	return nil
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
		conf.Hosts = append(conf.Hosts, ClientHostConfig{
			Name:          h.Name,
			Host:          h.Host,
			Token:         h.Token,
			SkipTLSVerify: h.SkipTLSVerify,
			ProviderID:    h.SourceID,
			Current:       h.Current,
		})
	}

	return conf
}
