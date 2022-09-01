package data

import (
	"github.com/infrahq/infra/internal/server/models"
)

func GetForgottenDomainsForEmail(tx ReadTxn, email string) ([]models.ForgottenDomain, error) {
	var results []models.ForgottenDomain

	rows, err := tx.Query("SELECT organizations.name, organizations.domain, identities.last_seen_at FROM identities, organizations WHERE identities.organization_id = organizations.id AND identities.name = ?", email)
	if err != nil {
		return results, err
	}
	defer rows.Close()

	for rows.Next() {
		var r models.ForgottenDomain
		if err := rows.Scan(&r.OrganizationName, &r.OrganizationDomain, &r.LastSeenAt); err != nil {
			return results, err
		}
		results = append(results, r)
	}

	if err = rows.Err(); err != nil {
		return results, err
	}
	return results, nil
}
