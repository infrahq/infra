package cliopts

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/pflag"
)

type FlagSet interface {
	VisitAll(fn func(*pflag.Flag))
}

// DefaultsFromEnv looks for an environment variable for any unset flags. When
// an environment variable is found, the flag value is set from the environment
// variable.
//
// The environment variable for a flag has the prefix prepended, dashes replaced
// with underscores, and lowercase converted to uppercase (ex:
// --my-flag would be set from PREFIX_MY_FLAG).
//
// DefaultsFromEnv should be called after FlagSet.Parse, but before any flags
// are used.
func DefaultsFromEnv(prefix string, flags FlagSet) error {
	replacer := strings.NewReplacer("-", "_")
	prefix = prefix + "_"

	var errs []error
	flags.VisitAll(func(flag *pflag.Flag) {
		if flag.Changed {
			return
		}

		key := strings.ToUpper(prefix + replacer.Replace(flag.Name))
		v, exists := os.LookupEnv(key)
		if !exists {
			return
		}
		if err := flag.Value.Set(v); err != nil {
			err = fmt.Errorf("failed to set %v from environment variable: %w", flag.Name, err)
			errs = append(errs, err)
		}
	})

	if len(errs) > 0 {
		return MultiError(errs)
	}
	return nil
}

type MultiError []error

func (e MultiError) Error() string {
	errs := ([]error)(e)
	switch len(errs) {
	case 1:
		return errs[0].Error()
	default:
		var sb strings.Builder
		sb.WriteString("multiple errors:")
		for _, err := range errs {
			sb.WriteString("\n    " + err.Error())
		}
		sb.WriteString("\n")
		return sb.String()
	}
}
