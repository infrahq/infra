package email

type PasswordResetData struct {
	Link string
}

func SendPasswordResetEmail(name, address string, data PasswordResetData) error {
	return SendTemplate(name, address, EmailTemplatePasswordReset, data)
}
