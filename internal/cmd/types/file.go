package types

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"syscall"

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
	pathError := &fs.PathError{}
	fh, err := os.Open(raw)
	switch {
	case errors.As(err, &pathError) && errors.Is(pathError.Err, syscall.ENAMETOOLONG):
		*s = StringOrFile(raw)
		return nil
	case errors.Is(err, os.ErrNotExist):
		// Only log a small prefix of the value, at trace level, in case the value is sensitive.
		logging.L.Trace().
			Str("valuePrefix", raw[:len(raw)/2]).
			Msg("value does not appear to be a file, assuming string literal")

		*s = StringOrFile(raw)
		return nil
	case err != nil:
		return err
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
