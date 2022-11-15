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

// clientConfigVersion is the current version of the client configuration file.
// Use this constant when referring to a value in tests or code that should
// always use latest.
var clientConfigVersion = ClientConfigVersion{Version: "0.4"}

type ClientConfigVersion struct {
	Version string `json:"version"`
}

type ClientConfig struct {
	ClientConfigVersion `json:",inline"`
	Hosts               []ClientHostConfig `json:"hosts"`
}

type ClientHostConfig struct {
	UserID        uid.ID   `json:"user-id"`
	Name          string   `json:"name"` // user name
	Host          string   `json:"host"`
	AccessKey     string   `json:"access-key,omitempty"`
	SkipTLSVerify bool     `json:"skip-tls-verify"` // where is the other cert info stored?
	ProviderID    uid.ID   `json:"provider-id,omitempty"`
	Expires       api.Time `json:"expires"`
	Current       bool     `json:"current"`
	// TrustedCertificate is the PEM encoded TLS certificate used by the server
	// that was verified and trusted by the user as part of login.
	TrustedCertificate string `json:"trusted-certificate"`
}

// checks if user is logged in to the given session (ClientHostConfig)
func (c *ClientHostConfig) isLoggedIn() bool {
	return c.AccessKey != "" && c.Name != "" && c.UserID != 0 && !c.isExpired()
}

func (c *ClientHostConfig) isExpired() bool {
	return time.Now().After(time.Time(c.Expires))
}

func infraHomeDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	infraDir := filepath.Join(homeDir, ".infra")

	return infraDir, nil
}

func initInfraHomeDir() (string, error) {
	infraDir, err := infraHomeDir()
	if err != nil {
		return "", err
	}

	if err := os.MkdirAll(infraDir, os.ModePerm); err != nil {
		return "", err
	}

	return infraDir, nil
}

func readConfig() (*ClientConfig, error) {
	infraDir, err := initInfraHomeDir()
	if err != nil {
		return nil, err
	}

	contents, err := ioutil.ReadFile(filepath.Join(infraDir, "config"))
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	if len(contents) == 0 {
		return &ClientConfig{ClientConfigVersion: clientConfigVersion}, nil
	}

	config := &ClientConfigVersion{}
	if err = json.Unmarshal(contents, &config); err != nil {
		return nil, err
	}

	switch config.Version {
	case "":
		// if version is missing, it needs to be updated to the latest
		configv0dot1 := ClientConfigV0dot1{}
		if err = json.Unmarshal(contents, &configv0dot1); err != nil {
			return nil, err
		}

		return configv0dot1.ToV0dot2().ToV0dot3().ToV0dot4(), nil

	case "0.2":
		configv0dot2 := ClientConfigV0dot2{}
		if err = json.Unmarshal(contents, &configv0dot2); err != nil {
			return nil, err
		}

		return configv0dot2.ToV0dot3().ToV0dot4(), nil
	case "0.3":
		configv0dot3 := ClientConfigV0dot3{}
		if err = json.Unmarshal(contents, &configv0dot3); err != nil {
			return nil, err
		}

		return configv0dot3.ToV0dot4(), nil
	case "0.4":
		config := &ClientConfig{}
		if err = json.Unmarshal(contents, &config); err != nil {
			return nil, err
		}
		return config, nil
	default:
		return nil, errors.New("unknown version " + config.Version)
	}
}

func writeConfig(config *ClientConfig) error {
	infraDir, err := initInfraHomeDir()
	if err != nil {
		return err
	}

	contents, err := json.Marshal(config)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(filepath.Join(infraDir, "config"), contents, 0o600)
}

// Save (create or update) the current hostconfig
func saveHostConfig(hostConfig ClientHostConfig) error {
	config, err := readConfig()
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

// Deprecated: use readConfig and ClientConfig.CurrentHostConfig
func currentHostConfig() (*ClientHostConfig, error) {
	cfg, err := readConfig()
	if err != nil {
		return nil, err
	}
	return cfg.CurrentHostConfig()
}

func (c ClientConfig) CurrentHostConfig() (*ClientHostConfig, error) {
	for i, host := range c.Hosts {
		if host.Current {
			return &c.Hosts[i], nil
		}
	}

	return nil, ErrConfigNotFound
}

func (c ClientConfig) APIClient() (*api.Client, error) {
	cfg, err := readConfig()
	if err != nil {
		return nil, err
	}
	hostConfig, err := cfg.CurrentHostConfig()
	if err != nil {
		return nil, err
	}
	return apiClientFromHostConfig(hostConfig)
}
