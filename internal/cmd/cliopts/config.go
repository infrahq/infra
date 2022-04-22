package cliopts

import (
	"fmt"
	"os"
	"strings"

	"github.com/mitchellh/mapstructure"
	"gopkg.in/yaml.v2"
)

type Options struct {
	FieldTagName string
	Filename     string
	EnvPrefix    string
	Flags        FlagSet
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
	if opts.FieldTagName == "" {
		opts.FieldTagName = "config"
	}
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

	if err := decode(target, raw, opts); err != nil {
		return fmt.Errorf("failed to decode from %s: %w", opts.Filename, err)
	}
	return nil
}

func decode(target interface{}, raw map[string]interface{}, opts Options) error {
	cfg := mapstructure.DecoderConfig{
		Squash:  true,
		Result:  target,
		TagName: opts.FieldTagName,
		MatchName: func(key string, fieldName string) bool {
			return strings.EqualFold(key, fieldName)
		},
		DecodeHook: mapstructure.ComposeDecodeHookFunc(
			mapstructure.StringToTimeDurationHookFunc(),
			mapstructure.StringToSliceHookFunc(","),
			HookPrepareForDecode,
			HookSetFromString,
		),
	}
	decoder, err := mapstructure.NewDecoder(&cfg)
	if err != nil {
		return err
	}
	return decoder.Decode(raw)
}
