package email

type UserInviteData struct {
	FromUserName string
	Link         string
}

func SendUserInviteEmail(name, address string, data UserInviteData) error {
	return SendTemplate(name, address, EmailTemplateUserInvite, data)
}
