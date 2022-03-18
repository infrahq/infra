package models

import "time"

// Token is presented at a resource managed by Infra (ex: an Infra connector) to assert claims
type Token struct {
	Token   string
	Expires time.Time
}
