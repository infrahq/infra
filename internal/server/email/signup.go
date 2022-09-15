package email

type SignupData struct {
	Link        string
	WrappedLink string
}

func SendSignupEmail(name, address string, data SignupData) error {
	return SendTemplate(name, address, EmailTemplateSignup, data, BypassListManagement)
}
