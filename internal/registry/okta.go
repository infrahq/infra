package registry

import (
	"context"
	"errors"
	"time"

	timer "github.com/infrahq/infra/internal/timer"
	"github.com/okta/okta-sdk-golang/v2/okta"
	"golang.org/x/oauth2"
	"gopkg.in/square/go-jose.v2/jwt"
)

type Okta interface {
	ValidateOktaConnection(domain string, clientID string, apiToken string) error
	Emails(domain string, clientID string, apiToken string) ([]string, error)
	Groups(domain string, clientID string, apiToken string) (map[string][]string, error)
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
	defer timer.LogTimeElapsed(time.Now(), "okta user sync")

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
		profile, ok := oktaUser.Profile.(map[string]interface{})
		if !ok {
			continue
		}

		email, ok := profile["email"].(string)
		if !ok {
			continue
		}

		emails = append(emails, email)
	}

	return emails, nil
}

// Groups retrieves groups that exist in Okta for the configured InfraHQ group-role mappings and returns a map of group names to user lists
func (o *oktaImplementation) Groups(domain string, clientID string, apiToken string) (map[string][]string, error) {
	defer timer.LogTimeElapsed(time.Now(), "okta group sync")

	ctx, client, err := okta.NewClient(context.TODO(), okta.WithOrgUrl("https://"+domain), okta.WithRequestTimeout(30), okta.WithRateLimitMaxRetries(3), okta.WithToken(apiToken))
	if err != nil {
		return nil, err
	}

	// this returns a list of group IDs assigned to our client, we next need to find which names these IDs correspond to
	oktaApplicationGroups, resp, err := client.Application.ListApplicationGroupAssignments(ctx, clientID, nil)
	if err != nil {
		return nil, err
	}

	for resp.HasNextPage() {
		var nextAppGroupSet []*okta.ApplicationGroupAssignment

		resp, err = resp.Next(ctx, &nextAppGroupSet)
		if err != nil {
			return nil, err
		}

		oktaApplicationGroups = append(oktaApplicationGroups, nextAppGroupSet...)
	}

	// we have an API token, so we can list all Okta groups to avoid needing to query for each ID to get the name for that group
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

	// get the ID to group name mapping for looking up groups from the assigned application groups
	grpNames := make(map[string]string)
	for _, oktaGroup := range oktaGroups {
		grpNames[oktaGroup.Id] = oktaGroup.Profile.Name
	}

	// for each group in the infra config that is assigned to the application, find the users it has in Okta
	grpUsers := make(map[string][]string)

	for _, g := range oktaApplicationGroups {
		gUsers, resp, err := client.Group.ListGroupUsers(ctx, g.Id, nil)
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

		emails := []string{}

		for _, gUser := range gUsers {
			profile := *gUser.Profile

			email, ok := profile["email"].(string)
			if !ok {
				continue
			}

			if email != "" {
				emails = append(emails, email)
			}
		}

		name := grpNames[g.Id]
		grpUsers[name] = emails
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

	raw, ok := exchanged.Extra("id_token").(string)
	if !ok {
		return "", errors.New("could not extract id_token from oauth2 token")
	}

	tok, err := jwt.ParseSigned(raw)
	if err != nil {
		return "", err
	}

	out := make(map[string]interface{})
	if err := tok.UnsafeClaimsWithoutVerification(&out); err != nil {
		return "", err
	}

	email, ok := out["email"].(string)
	if !ok {
		return "", errors.New("could not extract email from identity provider token")
	}

	return email, nil
}
