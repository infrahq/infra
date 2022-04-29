package authn

import (
	"fmt"
)

var (
	ErrValidation                  = fmt.Errorf("validation failed")
	ErrInvalidProviderURL          = fmt.Errorf("%w: invalid provider url", ErrValidation)
	ErrInvalidProviderClientID     = fmt.Errorf("%w: invalid provider client id", ErrValidation)
	ErrInvalidProviderClientSecret = fmt.Errorf("%w: invalid provider client secret", ErrValidation)
)
