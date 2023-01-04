package email

import (
	"sort"

	"github.com/infrahq/infra/internal/server/data"
)

type ForgottenDomainData struct {
	Organizations []data.ForgottenDomain
}

func SendForgotDomainsEmail(name, address string, data ForgottenDomainData) error {
	// TODO: this should probably be done in the caller, but we don't have a reasonable
	// way to test it in the caller. Once we can test email sends from the API handlers
	// consider moving this to the caller.
	sort.Slice(data.Organizations, func(i, j int) bool {
		return data.Organizations[i].LastSeenAt.After(data.Organizations[j].LastSeenAt)
	})
	return SendTemplate(name, address, EmailTemplateForgottenDomains, data, BypassListManagement)
}
