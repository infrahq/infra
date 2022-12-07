//go:build !windows

package cmd

import (
	"io"
)

func newStderr(w io.Writer) io.Writer {
	return w
}
