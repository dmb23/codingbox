package sandbox

import (
	"context"
	"io"
	"os"

	"github.com/moby/moby/client"
	"golang.org/x/term"
)

// AttachInteractive attaches stdin/stdout to the container's TTY.
// It sets the terminal to raw mode and restores it on return.
// It blocks until the container's output stream closes (i.e., the container exits).
func AttachInteractive(ctx context.Context, cli *client.Client, containerID string) error {
	resp, err := cli.ContainerAttach(ctx, containerID, client.ContainerAttachOptions{
		Stdin:  true,
		Stdout: true,
		Stderr: true,
		Stream: true,
	})
	if err != nil {
		return err
	}
	defer resp.Close()

	// Set terminal to raw mode for interactive use.
	fd := int(os.Stdin.Fd())
	if term.IsTerminal(fd) {
		oldState, err := term.MakeRaw(fd)
		if err != nil {
			return err
		}
		defer term.Restore(fd, oldState)
	}

	// Copy stdin to container in background.
	go func() {
		io.Copy(resp.Conn, os.Stdin)
		resp.CloseWrite()
	}()

	// Copy container output to stdout — blocks until container exits.
	_, err = io.Copy(os.Stdout, resp.Reader)
	return err
}
