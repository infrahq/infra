package server

import (
	"github.com/spf13/pflag"

	"github.com/infrahq/infra/internal/cmd/cliopts"
	"github.com/infrahq/infra/internal/logging"
)

type OptionsDiffV0dot1 struct {
	Identities []User `validate:"dive"`
}

// ToV0dot2 applies the 0.1 options to the 0.2 version
func (o OptionsDiffV0dot1) applyToV0dot2(base *Options) {
	logging.S.Warn("updated server options from version 0.1 to 0.2")

	base.Version = 0.2
	base.Config.Users = o.Identities
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

	if baseOpts.Version == 0 {
		// re-load the options into the correct version
		var optionsDiffV0dot1 OptionsDiffV0dot1
		err := cliopts.Load(&optionsDiffV0dot1, cliopts.Options{
			Filename:  configFilename,
			EnvPrefix: "INFRA_SERVER",
			Flags:     flags,
		})
		if err != nil {
			return err
		}

		optionsDiffV0dot1.applyToV0dot2(baseOpts)
	}

	return nil
}
