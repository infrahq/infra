package internal

import (
	"fmt"
)

var (
	ErrExpired   = fmt.Errorf("token expired")
	ErrInvalid   = fmt.Errorf("token invalid")
	ErrForbidden = fmt.Errorf("forbidden")

	ErrDuplicate = fmt.Errorf("duplicate record")
	ErrNotFound  = fmt.Errorf("record not found")
)
