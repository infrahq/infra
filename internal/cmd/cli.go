package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
)

// CLI exposes common dependencies to commands.
type CLI struct {
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
}

// Output a string to CLI.Stdout. Output is like fmt.Printf except that it always
// adds a trailing newline.
// To write output without a trailing newline use CLI.Stdout directly.
func (c *CLI) Output(format string, args ...interface{}) {
	fmt.Fprintf(c.Stdout, format+"\n", args...)
}

// key is a type to ensure no other package can access the CLI value in context.
type key struct{}

// ctxKey used to store CLI in the context.
var ctxKey = key{}

// newCli looks for a CLI stores in context. If one exists, the CLI from
// context is returned, otherwise a new CLI is created with streams set to the
// standard input and output streams.
//
// newCLI is a shim for testing, allowing tests to use a buffer instead of the
// standard streams.
func newCLI(ctx context.Context) *CLI {
	cli, ok := ctx.Value(ctxKey).(*CLI)
	if ok {
		return cli
	}
	return &CLI{
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}
}
