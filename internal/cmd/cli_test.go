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
		Stdin:  new(bytes.Buffer),
		Stdout: new(bytes.Buffer),
		Stderr: new(bytes.Buffer),
	}
	cli := &CLI{
		Stdout: writerWithFd{
			Writer: io.MultiWriter(bufs.Stdout, os.Stdout),
			fd:     os.Stdout.Fd(),
		},
		Stderr: io.MultiWriter(bufs.Stderr, os.Stderr),
		Stdin:  readerWithFd{Reader: bufs.Stdin, fd: os.Stdin.Fd()},
	}
	return context.WithValue(ctx, ctxKey, cli), bufs
}

type BufferedStreams struct {
	Stdin  *bytes.Buffer
	Stdout *bytes.Buffer
	Stderr *bytes.Buffer
}

type readerWithFd struct {
	io.Reader
	fd uintptr
}

func (r readerWithFd) Fd() uintptr {
	return r.fd
}

type writerWithFd struct {
	io.Writer
	fd uintptr
}

func (w writerWithFd) Fd() uintptr {
	return w.fd
}

func PatchCLIWithPTY(ctx context.Context, pty *os.File) context.Context {
	cli := &CLI{
		Stdout: pty,
		Stderr: pty,
		Stdin:  pty,
	}
	return context.WithValue(ctx, ctxKey, cli)
}
