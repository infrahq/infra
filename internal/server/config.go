package server

import (
	"errors"
	"sync/atomic"

	"github.com/infrahq/infra/internal/generate"
)

type Config struct {
	Providers struct {
		Okta struct {
			Domain       string `json:"domain" yaml:"domain,omitempty"`
			ClientID     string `json:"clientID" yaml:"clientID,omitempty"`
			ClientSecret string `json:"-" yaml:"clientSecret,omitempty"`
			ApiToken     string `json:"-" yaml:"apiToken,omitempty"`
		} `json:"okta" yaml:"okta,omitempty"`
	} `json:"providers" yaml:"providers,omitempty"`

	System struct {
		TokenSecret string `json:"tokenSecret" yaml:"tokenSecret"`
	} `json:"system" yaml:"system"`

	Permissions []struct {
		User string `json:"user" yaml:"user"`
		Role string `json:"role" yaml:"role"`
	} `yaml:"permissions,omitempty"`
}

type ConfigStore struct {
	v atomic.Value
}

func (cs *ConfigStore) get() *Config {
	return cs.v.Load().(*Config)
}

func (cs *ConfigStore) set(config *Config) {
	cs.v.Store(config)
}

func InitConfig(config *Config) {
	if config.System.TokenSecret == "" {
		config.System.TokenSecret = generate.RandString(32)
	}
}

// TODO(jmorganca): implement me by parsing yaml and validating against a struct schema
func ValidateConfig(config Config) error {
	return errors.New("not implemented")
}
