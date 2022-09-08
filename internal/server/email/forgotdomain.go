package email

import (
	"github.com/infrahq/infra/internal/server/models"
)

type ForgottenDomainData struct {
	Domains []models.ForgottenDomain
}

func SendForgotDomains(name, address string, data ForgottenDomainData) error {
	return SendTemplate(name, address, EmailTemplateForgottenDomains, data)
}
