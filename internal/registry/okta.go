package registry

import (
	"context"

	grpc_zap "github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"
	"github.com/okta/okta-sdk-golang/v2/okta"
	"golang.org/x/oauth2"
	"gopkg.in/square/go-jose.v2/jwt"
)

type Okta interface {
	ValidateOktaConnection(domain string, clientID string, apiToken string) error
	Emails(domain string, clientID string, apiToken string) ([]string, error)
	Groups(domain string, clientID string, apiToken string, groupNames []string) (map[string][]string, error)
	EmailFromCode(code string, domain string, clientID string, clientSecret string) (string, error)
}

type oktaImplementation struct{}

func NewOkta() Okta {
	return &oktaImplementation{}
}

// ValidateOktaConnection requests the client from Okta to check for errors on the response
func (o *oktaImplementation) ValidateOktaConnection(domain string, clientID string, apiToken string) error {
	_, _, err := okta.NewClient(context.TODO(), okta.WithOrgUrl("https://"+domain), okta.WithRequestTimeout(30), okta.WithRateLimitMaxRetries(3), okta.WithToken(apiToken))
	return err
}

func (o *oktaImplementation) Emails(domain string, clientID string, apiToken string) ([]string, error) {
	ctx, client, err := okta.NewClient(context.TODO(), okta.WithOrgUrl("https://"+domain), okta.WithRequestTimeout(30), okta.WithRateLimitMaxRetries(3), okta.WithToken(apiToken))
	if err != nil {
		return nil, err
	}

	oktaUsers, resp, err := client.Application.ListApplicationUsers(ctx, clientID, nil)
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

// Groups retrieves groups that exist in Okta for the configured InfraHQ group-role mappings and returns a map of group names to user lists
func (o *oktaImplementation) Groups(domain string, clientID string, apiToken string, groupNames []string) (map[string][]string, error) {
	ctx, client, err := okta.NewClient(context.TODO(), okta.WithOrgUrl("https://"+domain), okta.WithRequestTimeout(30), okta.WithRateLimitMaxRetries(3), okta.WithToken(apiToken))
	if err != nil {
		return nil, err
	}

	// we have an API token, so we can list all Okta groups and save the admin the step of linking them to the InfraHQ application
	oktaGroups, resp, err := client.Group.ListGroups(ctx, nil)
	if err != nil {
		return nil, err
	}
	for resp.HasNextPage() {
		var nextGroupSet []*okta.Group
		resp, err = resp.Next(ctx, &nextGroupSet)
		if err != nil {
			return nil, err
		}
		oktaGroups = append(oktaGroups, nextGroupSet...)
	}

	// get the IDs for groups so we can look up the users for the ones we care about
	grpIDs := make(map[string]string)
	for _, oktaGroup := range oktaGroups {
		grpIDs[oktaGroup.Profile.Name] = oktaGroup.Id
	}

	// for each group in the infra config, find the users it has in Okta
	grpUsers := make(map[string][]string)
	for _, g := range groupNames {
		id := grpIDs[g]
		if id == "" {
			grpc_zap.Extract(ctx).Debug("ignoring group that does not exist in okta: " + g)
			continue
		}

		gUsers, resp, err := client.Group.ListGroupUsers(ctx, id, nil)
		if err != nil {
			return nil, err
		}
		for resp.HasNextPage() {
			var nextUserSet []*okta.User
			resp, err = resp.Next(ctx, &nextUserSet)
			if err != nil {
				return nil, err
			}
			gUsers = append(gUsers, nextUserSet...)
		}

		var emails []string
		for _, gUser := range gUsers {
			profile := *gUser.Profile
			email := profile["email"].(string)
			if email != "" {
				emails = append(emails, email)
			}
		}
		grpUsers[g] = emails
	}
	return grpUsers, nil
}

func (o *oktaImplementation) EmailFromCode(code string, domain string, clientID string, clientSecret string) (string, error) {
	ctx := context.Background()
	conf := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  "http://localhost:8301",
		Scopes:       []string{"openid", "email"},
		Endpoint: oauth2.Endpoint{
			TokenURL: "https://" + domain + "/oauth2/v1/token",
			AuthURL:  "https://" + domain + "/oauth2/v1/authorize",
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
