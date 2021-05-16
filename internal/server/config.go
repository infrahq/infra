package server

import (
	"io/ioutil"
	"os"

	"gopkg.in/yaml.v2"
)

type OktaConfig struct {
	Domain       string `yaml:"domain" json:"domain"`
	ClientID     string `yaml:"client-id" json:"client-id"`
	ClientSecret string `yaml:"client-secret"` // TODO(jmorganca): move this to a secret
	ApiToken     string `yaml:"api-token"`     // TODO(jmorganca): move this to a secret
}

type ServerConfig struct {
	Providers struct {
		Okta OktaConfig `yaml:"okta" json:"okta"`
	}
	Permissions []struct {
		User       string
		Group      string
		Permission string
	}
}

func loadConfig(path string) (*ServerConfig, error) {
	contents, err := ioutil.ReadFile(path)
	if os.IsNotExist(err) {
		return &ServerConfig{}, nil
	}

	if err != nil {
		return nil, err
	}

	var config ServerConfig
	err = yaml.Unmarshal([]byte(contents), &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}
