package sandbox

import (
	"context"
	"io"
	"os"
	"os/signal"
	"syscall"

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

		// Forward host terminal resizes to the container TTY.
		go monitorResize(ctx, cli, containerID, fd)
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

// monitorResize listens for SIGWINCH and resizes the container TTY to match
// the host terminal. It runs until ctx is cancelled.
func monitorResize(ctx context.Context, cli *client.Client, containerID string, fd int) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGWINCH)
	defer signal.Stop(sigCh)

	for {
		select {
		case <-ctx.Done():
			return
		case <-sigCh:
			w, h, err := term.GetSize(fd)
			if err != nil {
				continue
			}
			cli.ContainerResize(ctx, containerID, client.ContainerResizeOptions{
				Height: uint(h),
				Width:  uint(w),
			})
		}
	}
}
