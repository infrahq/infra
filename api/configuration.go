package api

type ServerConfiguration struct {
	IsEmailConfigured bool      `json:"isEmailConfigured"`
	IsSignupEnabled   bool      `json:"isSignupEnabled"`
	BaseDomain        string    `json:"baseDomain"`
	LoginDomain       string    `json:"loginDomain"`
	Google            *Provider `json:"google"`
}
