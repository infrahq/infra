package email

type UserInviteData struct {
	FromUserName string
	Link         string
}

func SendUserInvite(name, address string, data UserInviteData) error {
	return SendTemplate(name, address, EmailTemplateUserInvite, map[string]interface{}{
		"fromName": data.FromUserName,
		"link":     data.Link,
	})
}
