package types

import (
	"io"
	"os"

	"github.com/infrahq/infra/internal/logging"
)

// StringOrFile is a pflag.Value type that can be used to read a value either
// from the command line flag, or as path to a file.
// If the value is not an existing filepath, it will be used as the literal
// string.
type StringOrFile string

func (s *StringOrFile) String() string {
	if s == nil {
		return ""
	}
	return string(*s)
}

func (s *StringOrFile) Set(raw string) error {
	fh, err := os.Open(raw)
	if err != nil {
		logging.L.Debug().Err(err).Msg("failed to open file")
		*s = StringOrFile(raw)
		return nil
	}

	content, err := io.ReadAll(fh)
	if err != nil {
		return err
	}
	*s = StringOrFile(content)
	return nil
}

func (s *StringOrFile) Type() string {
	return "filepath"
}
