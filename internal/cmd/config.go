package cmd

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/uid"
)

// current: v0.3
type ClientConfig struct {
	Version string             `json:"version"`
	Hosts   []ClientHostConfig `json:"hosts"`
}

// current: v0.3
type ClientHostConfig struct {
	PolymorphicID uid.PolymorphicID `json:"polymorphic-id"` // identity pid TODO: name this. what's it an ID of?
	Name          string            `json:"name"`           // identity name
	Host          string            `json:"host"`
	AccessKey     string            `json:"access-key,omitempty"`
	SkipTLSVerify bool              `json:"skip-tls-verify"` // where is the other cert info stored?
	ProviderID    uid.ID            `json:"provider-id,omitempty"`
	Expires       api.Time          `json:"expires"`
	Current       bool              `json:"current"`
}

// checks if user is logged in to the given session (ClientHostConfig)
func (c *ClientHostConfig) isLoggedIn() bool {
	return c.AccessKey != "" && c.Name != "" && c.PolymorphicID != ""
}

func (c *ClientHostConfig) isExpired() bool {
	return time.Now().After(time.Time(c.Expires))
}

// Retrieves client config if it exists, else instances a new one.
func readOrCreateClientConfig() (*ClientConfig, error) {
	config, err := readConfig()
	if err != nil && !errors.Is(err, ErrConfigNotFound) {
		return nil, err
	}

	if config == nil {
		return NewClientConfig(), nil
	}

	return config, nil
}

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
	return readHostConfig()
}

// Save (create or update) the current hostconfig
func saveHostConfig(hostConfig ClientHostConfig) error {
	config, err := readOrCreateClientConfig()
	if err != nil {
		return err
	}

	var found bool
	for i, c := range config.Hosts {
		if c.Host == hostConfig.Host {
			config.Hosts[i] = hostConfig
			found = true

			continue
		}
		config.Hosts[i].Current = false
	}
	if !found {
		config.Hosts = append(config.Hosts, hostConfig)
	}

	if err := writeConfig(config); err != nil {
		return err
	}

	return nil
}

func readHostConfig() (*ClientHostConfig, error) {
	cfg, err := readConfig()
	if err != nil {
		return nil, err
	}

	for i, c := range cfg.Hosts {
		if c.Current {
			return &cfg.Hosts[i], nil
		}
	}

	return nil, ErrConfigNotFound
}
