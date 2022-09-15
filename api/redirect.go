package api

type VerifyAndRedirectRequest struct {
	VerificationToken string `form:"vt"`
	Base64RedirectURL string `form:"r"`
}

type RedirectResponse struct {
	RedirectTo string `json:"-"`
}

// satisfies the isRedirect interface
func (r RedirectResponse) RedirectURL() string {
	return r.RedirectTo
}
