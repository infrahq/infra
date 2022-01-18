package models

import "time"

type Token struct {
	Token   string
	Expires time.Time
}
