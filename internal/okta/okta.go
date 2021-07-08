package okta

import (
	"context"

	"github.com/okta/okta-sdk-golang/v2/okta"
	"golang.org/x/oauth2"
	"gopkg.in/square/go-jose.v2/jwt"
)

// ValidateOktaConnection requests the client from Okta to check for errors on the response
func ValidateOktaConnection(domain string, clientID string, apiToken string) error {
	_, _, err := okta.NewClient(context.TODO(), okta.WithOrgUrl("https://"+domain), okta.WithRequestTimeout(30), okta.WithRateLimitMaxRetries(3), okta.WithToken(apiToken))
	return err
}

func Emails(domain string, clientID string, apiToken string) ([]string, error) {
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

func EmailFromCode(code string, domain string, clientID string, clientSecret string) (string, error) {
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
