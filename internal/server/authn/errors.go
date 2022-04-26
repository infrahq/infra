package authn

import "fmt"

var (
	ErrInvalidProviderURL          = fmt.Errorf("invalid provider url")
	ErrInvalidProviderClientID     = fmt.Errorf("invalid client id")
	ErrInvalidProviderClientSecret = fmt.Errorf("invalid client secret")
)
