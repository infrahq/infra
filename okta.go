package main

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/okta/okta-sdk-golang/v2/okta"
)

const (
	SYNC_INTERVAL_SECONDS = 10
)

type Okta struct {
	Domain     string
	ClientID   string
	ApiToken   string
	Data       *Data
	Kubernetes *Kubernetes // TODO: factor me out of this or use channels to signal
	stop       chan bool
}

func (o *Okta) Start() {
	ticker := time.NewTicker(SYNC_INTERVAL_SECONDS * time.Second)
	o.Sync()
	go func() {
		for {
			select {
			case <-ticker.C:
				if err := o.Sync(); err != nil {
					fmt.Println(err)
				}
			case <-o.stop:
				ticker.Stop()
				return
			}
		}
	}()
}

func (o *Okta) Stop() {
	o.stop <- true
}

func (o *Okta) Sync() error {
	ctx, client, err := okta.NewClient(context.TODO(), okta.WithOrgUrl("https://"+o.Domain), okta.WithRequestTimeout(30), okta.WithRateLimitMaxRetries(3), okta.WithToken(o.ApiToken))
	if err != nil {
		return err
	}

	oktaUsers, resp, err := client.Application.ListApplicationUsers(ctx, o.ClientID, nil)
	if err != nil {
		return err
	}

	for resp.HasNextPage() {
		var nextUserSet []*okta.AppUser
		resp, err = resp.Next(ctx, &nextUserSet)
		if err != nil {
			return err
		}
		oktaUsers = append(oktaUsers, nextUserSet...)
	}

	oktaEmails := map[string]bool{}

	for _, oktaUser := range oktaUsers {
		profile := oktaUser.Profile.(map[string]interface{})
		email := profile["email"].(string)
		oktaEmails[email] = true
	}

	// Add users
	for email := range oktaEmails {
		user, err := o.Data.FindUser(email)
		if err != nil {
			return err
		}

		if user == nil {
			user = &User{}
		}

		user.Email = email
		providers := user.Providers
		if len(providers) == 0 {
			user.Providers = []string{"okta"}
		} else {
			hasOkta := false
			for _, p := range user.Providers {
				if p == "okta" {
					hasOkta = true
				}
			}
			if !hasOkta {
				user.Providers = append(user.Providers, "okta")
				sort.Strings(user.Providers)
			}
		}

		o.Data.PutUser(user)
	}

	users, err := o.Data.ListUsers()
	if err != nil {
		return err
	}

	// delete users who aren't in okta
	for _, user := range users {
		if !oktaEmails[user.Email] {
			providers := []string{}
			for _, p := range user.Providers {
				if p != "okta" {
					providers = append(providers, p)
				}
			}
			user.Providers = providers
			if len(user.Providers) == 0 {
				o.Data.DeleteUser(user.ID)
			}
		}
	}

	// TODO: refactor me
	if err = UpdateKubernetesClusterRoleBindings(o.Data, o.Kubernetes); err != nil {
		return err
	}

	return nil
}
