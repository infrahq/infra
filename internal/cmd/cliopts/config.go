package cliopts

import (
	"fmt"
	"os"

	"github.com/mitchellh/mapstructure"
	"gopkg.in/yaml.v2"
)

type Options struct {
	Filename  string
	EnvPrefix string
	Flags     FlagSet
}

// Load configuration into target. Configuration may come from multiple sources.
//
// To set default values, apply them to target before calling Load.
// Configuration is loaded in the following order:
//    1. from a yaml file identified by opts.Filename
//    2. from environment variables that start with opts.EnvPrefix
//    3. from command line flags in opts.Flags
//
// Values are matched to the fields in target by convention. To override the
// convention use the 'config' struct field tag to specify a different name.
//
// For example, the field target.Addr.HTTPS would be set from:
//
//    // YAML
//   {"addr": {"https": "value"}}
//   // environment variable
//   PREFIX_ADDR_HTTPS=value
//   // command line flag
//   flags.String("addr-https", ...)
//
func Load(target interface{}, opts Options) error {
	if opts.Filename != "" {
		if err := loadFromFile(target, opts); err != nil {
			return err
		}
	}
	if opts.EnvPrefix != "" {
		if err := loadFromEnv(target, opts); err != nil {
			return err
		}
	}
	if opts.Flags != nil {
		if err := loadFromFlags(target, opts); err != nil {
			return err
		}
	}
	return nil
}

func loadFromFile(target interface{}, opts Options) error {
	fh, err := os.Open(opts.Filename)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	var raw map[string]interface{}
	if err := yaml.NewDecoder(fh).Decode(&raw); err != nil {
		return fmt.Errorf("failed to decode yaml from %s: %w", opts.Filename, err)
	}

	cfg := DecodeConfig(target)
	decoder, err := mapstructure.NewDecoder(&cfg)
	if err != nil {
		return err
	}
	if err := decoder.Decode(raw); err != nil {
		return fmt.Errorf("failed to decode from %s: %w", opts.Filename, err)
	}
	return nil
}

const fieldTagName = "config"

// DecodeConfig returns the default DecoderConfig used by Load. This config
// can be used by tests in other packages to simulate a call to Load.
func DecodeConfig(target interface{}) mapstructure.DecoderConfig {
	return mapstructure.DecoderConfig{
		Squash:  true,
		Result:  target,
		TagName: fieldTagName,
		DecodeHook: mapstructure.ComposeDecodeHookFunc(
			mapstructure.StringToTimeDurationHookFunc(),
			mapstructure.StringToSliceHookFunc(","),
			hookFlagValueSlice,
			hookPrepareForDecode,
			hookSetFromString,
		),
	}
}
