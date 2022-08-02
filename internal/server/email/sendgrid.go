package email

import (
	"errors"
	"os"

	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"

	"github.com/infrahq/infra/internal/logging"
)

type EmailTemplate int8

const (
	EmailTemplateAccountCreated EmailTemplate = iota
	EmailTemplatePasswordReset
	EmailTemplateUserInvite
)

var emailTemplateIDs = map[EmailTemplate]string{
	EmailTemplateAccountCreated: "",
	EmailTemplatePasswordReset:  "d-d87873d2f11b4055befbd7064cda44d6", // transactional-password-reset
	EmailTemplateUserInvite:     "d-ac6bcb2a7f02463c8b7fc8caffc35f2d", // transactional-user-invite // TODO: Should this instead be a "granted access to x" email?
}

var (
	AppDomain          = "https://infrahq.com"
	FromAddress        = "noreply@infrahq.com"
	FromName           = "Infra"
	SendgridAPIKey     = os.Getenv("SENDGRID_API_KEY")
	TestMode           = false
	TestDataSent       = []map[string]interface{}{}
	ErrUnknownTemplate = errors.New("unknown template")
	ErrNotConfigured   = errors.New("email sending not configured")
)

func IsConfigured() bool {
	return len(SendgridAPIKey) > 0
}

func SendTemplate(name, address string, template EmailTemplate, data map[string]interface{}) error {
	if TestMode {
		logging.Debugf("sent email to %q: %+v\n", address, data)
		TestDataSent = append(TestDataSent, data)
		return nil // quietly return
	}

	if len(SendgridAPIKey) == 0 {
		return ErrNotConfigured
	}

	m := mail.NewV3Mail()

	m.SetFrom(mail.NewEmail(FromName, FromAddress))

	templateID, ok := emailTemplateIDs[template]
	if !ok || len(templateID) == 0 {
		return ErrUnknownTemplate
	}

	m.SetTemplateID(templateID)

	p := mail.NewPersonalization()
	p.AddTos([]*mail.Email{mail.NewEmail(name, address)}...)

	for k, v := range data {
		p.SetDynamicTemplateData(k, v)
	}

	m.AddPersonalizations(p)

	request := sendgrid.GetRequest(SendgridAPIKey, "/v3/mail/send", "https://api.sendgrid.com")
	request.Method = "POST"
	var Body = mail.GetRequestBody(m)
	request.Body = Body
	response, err := sendgrid.API(request)
	if response.StatusCode != 200 {
		logging.Debugf("sendgrid api responded with status code %d", response.StatusCode)
	}
	// TODO: handle rate limiting and send retries
	return err
}
