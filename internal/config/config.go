package config

import (
	"fmt"

	"github.com/go-playground/validator/v10"
	"github.com/infrahq/infra/internal/api"
	"github.com/infrahq/infra/uid"
)

type Provider struct {
	Name         string `yaml:"name" validate:"required"`
	URL          string `yaml:"url" validate:"required"`
	ClientID     string `yaml:"clientID" validate:"required"`
	ClientSecret string `yaml:"clientSecret" validate:"required"`
}

type Grant struct {
	User     string `yaml:"user" validate:"excluded_with=Group"`
	Group    string `yaml:"group" validate:"excluded_with=User"`
	Provider string `yaml:"provider"`
	Role     string `yaml:"role" validate:"required"`
	Resource string `yaml:"resource" validate:"required"`
}

type Config struct {
	Providers []Provider `yaml:"providers"`
	Grants    []Grant    `yaml:"grants"`
}

func Import(c *api.Client, config Config, replace bool) error {
	if err := validator.New().Struct(config); err != nil {
		return err
	}

	keep := make(map[uid.ID]bool)

	for _, p := range config.Providers {
		providers, err := c.ListProviders(p.Name)
		if err != nil {
			return err
		}

		var provider *api.Provider
		if len(providers) > 0 {
			provider, err = c.UpdateProvider(api.UpdateProviderRequest{
				ID:           providers[0].ID,
				Name:         p.Name,
				URL:          p.URL,
				ClientID:     p.ClientID,
				ClientSecret: p.ClientSecret,
			})
			if err != nil {
				return err
			}
		} else {
			provider, err = c.CreateProvider(&api.CreateProviderRequest{
				Name:         p.Name,
				URL:          p.URL,
				ClientID:     p.ClientID,
				ClientSecret: p.ClientSecret,
			})
			if err != nil {
				return err
			}
		}

		keep[provider.ID] = true
	}

	providers, err := c.ListProviders("")
	if err != nil {
		return err
	}

	if config.Providers != nil && replace {
		for _, p := range providers {
			if !keep[p.ID] {
				err := c.DeleteProvider(p.ID)
				if err != nil {
					return err
				}
			}
		}
	}

	keep = make(map[uid.ID]bool)

	for _, g := range config.Grants {
		if g.Provider == "" && len(providers) > 1 {
			return fmt.Errorf("error importing grant %s - %s - %s: provider must be specified", g.User+g.Group, g.Role, g.Resource)
		}

		if len(providers) == 0 {
			return fmt.Errorf("error importing grant %s - %s - %s: no providers configured", g.User+g.Group, g.Role, g.Resource)
		}

		provider := providers[0]

		for _, p := range providers {
			if p.Name == g.Provider {
				provider = p
			}
		}

		// create user if it doesn't exist
		var identityID string
		if g.User != "" {
			users, err := c.ListUsers(api.ListUsersRequest{Email: g.User})
			if err != nil {
				return fmt.Errorf("error importing grant: %w", err)
			}

			var user *api.User
			if len(users) == 0 {
				user, err = c.CreateUser(&api.CreateUserRequest{
					Email:      g.User,
					ProviderID: provider.ID,
				})
				if err != nil {
					return err
				}
			} else {
				user = &users[0]
			}

			identityID = "u:" + user.ID.String()
		}

		// create group if it doesn't exist
		if g.Group != "" {
			groups, err := c.ListGroups(api.ListGroupsRequest{Name: g.Group, ProviderID: provider.ID})
			if err != nil {
				return fmt.Errorf("error importing grant: %w", err)
			}

			var group *api.Group
			if len(groups) == 0 {
				group, err = c.CreateGroup(&api.CreateGroupRequest{
					Name:       g.Group,
					ProviderID: provider.ID,
				})
				if err != nil {
					return err
				}
			} else {
				group = &groups[0]
			}

			identityID = "g:" + group.ID.String()
		}

		// create grant if it doesn't exist
		grants, err := c.ListGrants(api.ListGrantsRequest{
			Identity:  identityID,
			Resource:  g.Resource,
			Privilege: g.Role,
		})
		if err != nil {
			return fmt.Errorf("error importing grant: %w", err)
		}

		var grant *api.Grant
		if len(grants) == 0 {
			grant, err = c.CreateGrant(&api.CreateGrantRequest{
				Identity:  identityID,
				Resource:  g.Resource,
				Privilege: g.Role,
			})
			if err != nil {
				return fmt.Errorf("error importing grant: %w", err)
			}
		} else {
			grant = &grants[0]
		}

		keep[grant.ID] = true
	}

	grants, err := c.ListGrants(api.ListGrantsRequest{})
	if err != nil {
		return err
	}

	if config.Grants != nil && replace {
		for _, g := range grants {
			if !keep[g.ID] {
				err := c.DeleteGrant(g.ID)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}
