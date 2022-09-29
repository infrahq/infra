package data

import (
	"time"

	"github.com/infrahq/infra/internal/format"
	"github.com/infrahq/infra/internal/server/models"
)

func GetForgottenDomainsForEmail(tx ReadTxn, email string) ([]models.ForgottenDomain, error) {
	var results []models.ForgottenDomain

	rows, err := tx.Query("SELECT organizations.name, organizations.domain, identities.last_seen_at FROM identities, organizations WHERE identities.organization_id = organizations.id AND identities.name = ? AND identities.deleted_at IS NULL AND organizations.deleted_at IS NULL", email)
	if err != nil {
		return results, err
	}
	defer rows.Close()

	for rows.Next() {
		var r models.ForgottenDomain
		var lastSeenAt time.Time
		if err := rows.Scan(&r.OrganizationName, &r.OrganizationDomain, &lastSeenAt); err != nil {
			return results, err
		}
		r.LastSeenAt = format.HumanTimeWithCase(lastSeenAt, "never", false)
		results = append(results, r)
	}

	if err = rows.Err(); err != nil {
		return results, err
	}
	return results, nil
}
