package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/AlecAivazis/survey/v2/terminal"

	"github.com/infrahq/infra/api"
)

// CLI exposes common dependencies to commands.
type CLI struct {
	Stdin  terminal.FileReader
	Stdout terminal.FileWriter
	Stderr io.Writer

	RootOptions RootOptions
}

type RootOptions struct {
	LogLevel            string
	SkipAPIVersionCheck bool
}

// Output a string to CLI.Stdout. Output is like fmt.Printf except that it always
// adds a trailing newline.
// To write output without a trailing newline use CLI.Stdout directly.
func (c *CLI) Output(format string, args ...interface{}) {
	fmt.Fprintf(c.Stdout, format+"\n", args...)
}

func (c *CLI) surveyIO(options *survey.AskOptions) error {
	options.Stdio.In = c.Stdin
	options.Stdio.Out = c.Stdout
	options.Stdio.Err = c.Stderr
	return nil
}

// key is a type to ensure no other package can access the CLI value in context.
type key struct{}

// ctxKey used to store CLI in the context.
var ctxKey = key{}

func (c *CLI) apiClient() (*api.Client, error) {
	opts, err := defaultClientOpts()
	if err != nil {
		return nil, err
	}
	client, err := NewAPIClient(opts)
	if err != nil {
		return nil, err
	}
	if err := c.serverCompatible(client); err != nil {
		return nil, err
	}

	return client, nil
}

func (c *CLI) serverCompatible(client *api.Client) error {
	if !c.RootOptions.SkipAPIVersionCheck && !strings.HasSuffix(client.URL, ".infrahq.com") {
		if err := api.ServerCompatible(context.Background(), client); err != nil {
			return fmt.Errorf("%w, download the CLI version that matches your server, reference https://infrahq.com/docs/reference/self-hosting#cli-versions", err)
		}
	}

	return nil
}

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
