package email

import (
	"github.com/infrahq/infra/internal/server/models"
)

type ForgottenDomainData struct {
	Organizations []models.ForgottenDomain
}

func SendForgotDomainsEmail(name, address string, data ForgottenDomainData) error {
	return SendTemplate(name, address, EmailTemplateForgottenDomains, data, BypassListManagement)
}
