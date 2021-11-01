package cmd

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
)

// current: v0.3
type ClientConfig struct {
	Version string             `json:"version"`
	Hosts   []ClientHostConfig `json:"hosts"`
}

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
