package cmd

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
)

type ClientConfigV1dot0 struct {
	Version    string           `json:"version"` // 1.0
	Registries []RegistryConfig `json:"registries"`
}

type RegistryConfig struct {
	Name          string `json:"name"`
	Host          string `json:"host"`
	Token         string `json:"token"`
	SkipTLSVerify bool   `json:"skip-tls-verify"` // where is the other cert info stored?
	SourceID      string `json:"source-id"`
	Current       bool   `json:"current"`
}

func NewClientConfig() *ClientConfigV1dot0 {
	return &ClientConfigV1dot0{
		Version: "1.0",
	}
}

func readConfig() (*ClientConfigV1dot0, error) {
	config := &ClientConfigV1dot0{}

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
		return configv0dot1.ToV1dot0(), nil
	}

	return config, nil
}

func writeConfig(config *ClientConfigV1dot0) error {
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

func readCurrentConfig() (*RegistryConfig, error) {
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
