package cmd

import (
	"encoding/json"
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

func NewClientConfig() *ClientConfigV0dot2 {
	return &ClientConfigV0dot2{
		Version: "0.2",
	}
}

func readConfig() (*ClientConfigV0dot2, error) {
	config := &ClientConfigV0dot2{}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	contents, err := ioutil.ReadFile(filepath.Join(homeDir, ".infra", "config"))
	if os.IsNotExist(err) {
		return nil, &ErrUnauthenticated{}
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
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	if err = os.MkdirAll(filepath.Join(homeDir, ".infra"), os.ModePerm); err != nil {
		return err
	}

	contents, err := json.Marshal(config)
	if err != nil {
		return err
	}

	if err = ioutil.WriteFile(filepath.Join(homeDir, ".infra", "config"), []byte(contents), 0o600); err != nil {
		return err
	}

	return nil
}

func removeConfig() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	err = os.Remove(filepath.Join(homeDir, ".infra", "config"))
	if err != nil {
		return err
	}

	return nil
}

func readCurrentConfig() (*ClientRegistryConfig, error) {
	cfg, err := readConfig()
	if err != nil {
		return nil, err
	}

	for i := range cfg.Registries {
		if cfg.Registries[i].Current {
			return &cfg.Registries[i], nil
		}
	}

	return nil, nil
}
