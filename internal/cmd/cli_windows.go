package cmd

import (
	"io"

	"github.com/muesli/termenv"
)

type ansiConsole struct {
	io.Writer
}

func newStderr(w io.Writer) io.Writer {
	return ansiConsole{w}
}

func (a ansiConsole) Write(p []byte) (int, error) {
	mode, err := termenv.EnableWindowsANSIConsole()
	if err != nil {
		return 0, err
	}
	defer termenv.RestoreWindowsConsole(mode)
	return a.Writer.Write(p)
}
