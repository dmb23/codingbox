package sandbox

import (
	"context"
	"fmt"

	"github.com/moby/moby/client"
)

// CreateNetwork creates a Docker bridge network for the sandbox session.
func CreateNetwork(ctx context.Context, cli *client.Client, name string) (string, error) {
	resp, err := cli.NetworkCreate(ctx, name, client.NetworkCreateOptions{
		Driver: "bridge",
	})
	if err != nil {
		return "", fmt.Errorf("creating network %q: %w", name, err)
	}
	return resp.ID, nil
}

// RemoveNetwork disconnects all containers and removes the network.
func RemoveNetwork(ctx context.Context, cli *client.Client, networkID string) error {
	info, err := cli.NetworkInspect(ctx, networkID, client.NetworkInspectOptions{})
	if err != nil {
		// Network may already be removed.
		return nil
	}

	for containerID := range info.Network.Containers {
		_, _ = cli.NetworkDisconnect(ctx, networkID, client.NetworkDisconnectOptions{
			Container: containerID,
			Force:     true,
		})
	}

	if _, err := cli.NetworkRemove(ctx, networkID, client.NetworkRemoveOptions{}); err != nil {
		return fmt.Errorf("removing network %q: %w", networkID, err)
	}
	return nil
}
