package cmd

import (
	"bytes"
	"context"
	"io"
	"os"
)

// PatchCLI returns a context which contains a CLI value with the output streams
// set to buffers. PatchCLI is used by tests to record the output produced by
// CLI commands.
func PatchCLI(ctx context.Context) (context.Context, BufferedStreams) {
	bufs := BufferedStreams{
		Stdout: new(bytes.Buffer),
		Stderr: new(bytes.Buffer),
	}
	cli := &CLI{
		Stdout: io.MultiWriter(bufs.Stdout, os.Stdout),
		Stderr: io.MultiWriter(bufs.Stderr, os.Stderr),
		Stdin:  os.Stdin,
	}
	return context.WithValue(ctx, ctxKey, cli), bufs
}

type BufferedStreams struct {
	Stdout *bytes.Buffer
	Stderr *bytes.Buffer
}
