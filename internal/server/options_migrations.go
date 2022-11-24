package server

import (
	"fmt"
	"time"

	"github.com/spf13/pflag"

	"github.com/infrahq/infra/internal/cmd/cliopts"
	"github.com/infrahq/infra/internal/logging"
)

type OptionsDiffV0dot1 struct {
	Identities []User
}

// ToV0dot2 applies the 0.1 options to the 0.2 version
func (o OptionsDiffV0dot1) applyToV0dot2(base *Options) {
	logging.Warnf("updated server options from version 0.1 to 0.2")

	base.Version = 0.2
	base.Config.Users = o.Identities
}

type OptionsDiffV0dot2 struct {
	SessionExtensionDeadline time.Duration // deprecated in v0.18.1, use SessionInactivityTimout instead
}

// ToV0dot2 applies the 0.2 options to the 0.3 version
func (o OptionsDiffV0dot2) applyToV0dot3(base *Options) {
	logging.Warnf("updated server options from version 0.2 to 0.3")

	base.Version = 0.3

	if o.SessionExtensionDeadline != 0 {
		base.SessionInactivityTimeout = o.SessionExtensionDeadline
	}
}

// ApplyOptions loads and migrates specified server options from a file
func ApplyOptions(baseOpts *Options, configFilename string, flags *pflag.FlagSet) error {
	err := cliopts.Load(baseOpts, cliopts.Options{
		Filename:  configFilename,
		EnvPrefix: "INFRA_SERVER",
		Flags:     flags,
	})
	if err != nil {
		return err
	}

	// re-load the options into the correct version
	if baseOpts.Version == 0 {
		var optionsDiffV0dot1 OptionsDiffV0dot1
		err := cliopts.Load(&optionsDiffV0dot1, cliopts.Options{
			Filename:  configFilename,
			EnvPrefix: "INFRA_SERVER",
			Flags:     flags,
		})
		if err != nil {
			return fmt.Errorf("migrate server options 0.1: %w", err)
		}

		optionsDiffV0dot1.applyToV0dot2(baseOpts)
	}
	if baseOpts.Version == 0.2 {
		var optionsDiffV0dot2 OptionsDiffV0dot2
		err := cliopts.Load(&optionsDiffV0dot2, cliopts.Options{
			Filename:  configFilename,
			EnvPrefix: "INFRA_SERVER",
			Flags:     flags,
		})
		if err != nil {
			return fmt.Errorf("migrate server options 0.2: %w", err)
		}

		optionsDiffV0dot2.applyToV0dot3(baseOpts)
	}

	return nil
}
