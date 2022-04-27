package authn

import (
	"fmt"

	"github.com/infrahq/infra/internal"
)

var (
	ErrInvalidProviderURL          = fmt.Errorf("%w: %s", internal.ErrBadRequest, "invalid provider url")
	ErrInvalidProviderClientID     = fmt.Errorf("%w: %s", internal.ErrBadRequest, "invalid provider client id")
	ErrInvalidProviderClientSecret = fmt.Errorf("%w: %s", internal.ErrBadRequest, "invalid provider client secret")
)
