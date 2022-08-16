package api

type ServerConfiguration struct {
	IsEmailConfigured bool `json:"isEmailConfigured"`
	IsSignupEnabled   bool `json:"isSignupEnabled"`
}
