package email

type SignupData struct {
	Link string
}

func SendSignupEmail(name, address string, data SignupData) error {
	return SendTemplate(name, address, EmailTemplateSignup, data)
}
