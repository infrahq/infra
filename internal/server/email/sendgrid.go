package email

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"strings"
	texttemplate "text/template"

	"github.com/ssoroka/slice"

	"github.com/infrahq/infra/internal/logging"
)

type EmailTemplate int8

const (
	EmailTemplateSignup EmailTemplate = iota
	EmailTemplatePasswordReset
	EmailTemplateUserInvite
	EmailTemplateForgottenDomains
)

type TemplateDetail struct {
	TemplateName string
	Subject      string
}

var emailTemplates = map[EmailTemplate]TemplateDetail{
	EmailTemplateSignup: {
		TemplateName: "signup",
		Subject:      "Welcome to Infra!",
	},
	EmailTemplatePasswordReset: {
		TemplateName: "password-reset",
		Subject:      "Password Reset",
	},
	EmailTemplateUserInvite: {
		TemplateName: "user-invite",
		Subject:      "{{.FromUserName}} has invited you to Infra",
	},
	EmailTemplateForgottenDomains: {
		TemplateName: "forgot-domain",
		Subject:      "Your sign-in links",
	},
}

var (
	AppDomain          = "https://infrahq.com"
	FromAddress        = "noreply@infrahq.com"
	FromName           = "Infra"
	SendgridAPIKey     = os.Getenv("SENDGRID_API_KEY")
	SMTPServer         = "smtp.sendgrid.net:465"
	TestMode           = false
	TestDataSent       = []any{}
	ErrUnknownTemplate = errors.New("unknown template")
	ErrNotConfigured   = errors.New("email sending not configured")
)

func IsConfigured() bool {
	return len(SendgridAPIKey) > 0
}

func SendTemplate(name, address string, template EmailTemplate, data any) error {
	details, ok := emailTemplates[template]
	if !ok {
		return ErrUnknownTemplate
	}

	t, err := texttemplate.New("subject").Parse(details.Subject)
	if err != nil {
		return fmt.Errorf("parsing subject: %w", err)
	}

	w := &bytes.Buffer{}
	if err := t.Execute(w, data); err != nil {
		return fmt.Errorf("rendering subject: %w", err)
	}

	msg := Message{
		FromName:    FromName,
		FromAddress: FromAddress,
		ToName:      name,
		ToAddress:   address,
		Subject:     w.String(),
	}

	// render template with "data"
	w.Reset()
	err = textTemplateList.ExecuteTemplate(w, details.TemplateName+".text.plain", data)
	if err != nil {
		return err
	}
	msg.PlainBody = w.Bytes()

	w = &bytes.Buffer{}
	err = htmlTemplateList.ExecuteTemplate(w, details.TemplateName+".text.html", data)
	if err != nil {
		return err
	}
	msg.HTMLBody = w.Bytes()

	if TestMode {
		logging.Debugf("sent email to %q: %+v\n", address, data)
		logging.Debugf("plain: %s", string(msg.PlainBody))
		logging.Debugf("html: %s", string(msg.HTMLBody))
		TestDataSent = append(TestDataSent, data)
		return nil // quietly return
	}

	if len(SendgridAPIKey) == 0 {
		return ErrNotConfigured
	}

	if name == "" {
		// until we have real user names
		name = BuildNameFromEmail(address)
	}

	// TODO: handle rate limiting, retries, understanding which errors are retryable, send queues, whatever
	if err := SendSMTP(msg); err != nil {
		logging.Errorf("SMTP mail delivery error: %s", err)
		return err
	}

	return nil
}

func BuildNameFromEmail(email string) (name string) {
	name = strings.Join(slice.Map[string, string](strings.Split(strings.Split(email, "@")[0], "."), func(s string) string {
		return strings.ToUpper(s[0:1]) + s[1:]
	}), " ")
	if name == "Mail" {
		name = "Admin"
	}
	return name
}
