package cmd

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
)

type ClientConfigV0dot2 struct {
	Version    string                 `json:"version"` // always 0.2 in v0.2
	Registries []ClientRegistryConfig `json:"registries"`
}

type ClientRegistryConfig struct {
	Name          string `json:"name"`
	Host          string `json:"host"`
	Token         string `json:"token"`
	SkipTLSVerify bool   `json:"skip-tls-verify"` // where is the other cert info stored?
	SourceID      string `json:"source-id"`
	Current       bool   `json:"current"`
}

var ErrConfigNotFound = errors.New("Could not read local credentials. Are you logged in? Use \"infra login\" to login.")

func NewClientConfig() *ClientConfigV0dot2 {
	return &ClientConfigV0dot2{
		Version: "0.2",
	}
}

func readConfig() (*ClientConfigV0dot2, error) {
	config := &ClientConfigV0dot2{}

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

		return configv0dot1.ToV0dot2(), nil
	}

	return config, nil
}

func writeConfig(config *ClientConfigV0dot2) error {
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

func currentRegistryConfig() (*ClientRegistryConfig, error) {
	return readRegistryConfig("")
}

func readRegistryConfig(registry string) (*ClientRegistryConfig, error) {
	cfg, err := readConfig()
	if err != nil {
		return nil, err
	}

	for i, c := range cfg.Registries {
		if len(registry) == 0 && c.Current {
			return &cfg.Registries[i], nil
		}

		if c.Host == registry {
			return &cfg.Registries[i], nil
		}
	}

	return nil, nil
}

func removeRegistryConfig(registry string) error {
	cfg, err := readConfig()
	if err != nil {
		return err
	}

	for i, c := range cfg.Registries {
		if c.Host == registry {
			cfg.Registries = append(cfg.Registries[:i], cfg.Registries[i+1:]...)
			break
		}
	}

	err = writeConfig(cfg)
	if err != nil {
		return err
	}

	return nil
}
