package models

import (
	"time"
)

type ForgottenDomain struct {
	OrganizationName   string
	OrganizationDomain string
	LastSeenAt         time.Time
}
