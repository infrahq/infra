package server

import (
	"context"
	"io/ioutil"

	"github.com/okta/okta-sdk-golang/v2/okta"
	"github.com/square/go-jose/jwt"
	"golang.org/x/oauth2"
	"gopkg.in/yaml.v2"
)

type OktaConfig struct {
	Domain       string `yaml:"domain" json:"domain"`
	ClientID     string `yaml:"client-id" json:"client-id"`
	ClientSecret string `yaml:"client-secret" json:"-"`
	ApiToken     string `yaml:"api-token" json:"-"`
}

type ProviderConfig struct {
	Okta OktaConfig `yaml:"okta"`
}

type Permission struct {
	User       string `yaml:"user"`
	Permission string `yaml:"permission"`
}

type Config struct {
	Providers   ProviderConfig `yaml:"providers"`
	Permissions []Permission   `yaml:"permissions"`
}

func NewConfig(path string) (config *Config, err error) {
	config = &Config{}

	contents, err := ioutil.ReadFile(path)
	if err != nil {
		return
	}

	err = yaml.Unmarshal([]byte(contents), &config)
	if err != nil {
		return
	}

	if config.Providers.Okta.ClientSecret != "" {
		bytes, err := ioutil.ReadFile(config.Providers.Okta.ClientSecret)
		if err != nil {
			return nil, err
		}
		config.Providers.Okta.ClientSecret = string(bytes)
	}

	if config.Providers.Okta.ApiToken != "" {
		bytes, err := ioutil.ReadFile(config.Providers.Okta.ApiToken)
		if err != nil {
			return nil, err
		}
		config.Providers.Okta.ApiToken = string(bytes)
	}

	return
}

func (o *OktaConfig) Valid() bool {
	return o.ClientID != "" && o.Domain != ""
}

func (o *OktaConfig) Emails() ([]string, error) {
	ctx, client, err := okta.NewClient(context.TODO(), okta.WithOrgUrl("https://"+o.Domain), okta.WithRequestTimeout(30), okta.WithRateLimitMaxRetries(3), okta.WithToken(o.ApiToken))
	if err != nil {
		return nil, err
	}

	oktaUsers, resp, err := client.Application.ListApplicationUsers(ctx, o.ClientID, nil)
	if err != nil {
		return nil, err
	}

	for resp.HasNextPage() {
		var nextUserSet []*okta.AppUser
		resp, err = resp.Next(ctx, &nextUserSet)
		if err != nil {
			return nil, err
		}
		oktaUsers = append(oktaUsers, nextUserSet...)
	}

	emails := []string{}

	for _, oktaUser := range oktaUsers {
		profile := oktaUser.Profile.(map[string]interface{})
		email := profile["email"].(string)
		emails = append(emails, email)
	}

	return emails, nil
}

func (o *OktaConfig) EmailFromCode(code string) (string, error) {
	ctx := context.Background()
	conf := &oauth2.Config{
		ClientID:     o.ClientID,
		ClientSecret: o.ClientSecret,
		RedirectURL:  "http://localhost:8301",
		Scopes:       []string{"openid", "email"},
		Endpoint: oauth2.Endpoint{
			TokenURL: "https://" + o.Domain + "/oauth2/v1/token",
			AuthURL:  "https://" + o.Domain + "/oauth2/v1/authorize",
		},
	}

	exchanged, err := conf.Exchange(ctx, code)
	if err != nil {
		return "", err
	}

	raw := exchanged.Extra("id_token").(string)
	tok, err := jwt.ParseSigned(raw)
	if err != nil {
		return "", err
	}

	out := make(map[string]interface{})
	tok.UnsafeClaimsWithoutVerification(&out)
	email := out["email"].(string)

	return email, nil
}

var PermissionOrdering = []string{"view", "edit", "admin"}

func IsEqualOrHigherPermission(a string, b string) bool {
	indexa := 0
	indexb := 0

	for i, p := range PermissionOrdering {
		if a == p {
			indexa = i
		}

		if b == p {
			indexb = i
		}
	}

	return indexa >= indexb
}

// Gets users permissions from config, with a catch-all of view
// TODO (jmorganca): should this be nothing instead of view?
func PermissionForEmail(email string, cfg *Config) string {
	for _, p := range cfg.Permissions {
		if p.User == email {
			return p.Permission
		}
	}

	// Default to view
	return "view"
}
