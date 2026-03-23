package sandbox

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/mount"
	"github.com/moby/moby/api/types/network"
	"github.com/moby/moby/client"

	"github.com/mischa/codingbox/internal/models"
	"github.com/mischa/codingbox/internal/proxy"
	"github.com/mischa/codingbox/internal/store"
)

// Manager handles the sandbox lifecycle.
type Manager struct {
	cli    *client.Client
	sb     *models.Sandbox
	proxy  *proxy.Proxy
	store  *store.Store
	cancel context.CancelFunc
}

// NewManager creates a new sandbox manager.
func NewManager(cfg *models.SandboxConfig) (*Manager, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("connecting to Docker: %w", err)
	}

	sb := &models.Sandbox{
		ID:        uuid.New().String()[:8],
		Config:    *cfg,
		State:     models.StateCreating,
		CreatedAt: time.Now(),
	}

	return &Manager{cli: cli, sb: sb}, nil
}

// Start creates the network, container, and attaches an interactive session.
// It blocks until the session ends.
func (m *Manager) Start(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	m.cancel = cancel
	defer m.Stop()

	// Trap signals for graceful shutdown.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		select {
		case <-sigCh:
			cancel()
		case <-ctx.Done():
		}
	}()

	// 1. Start proxy and store.
	configDir := proxy.ConfigDir()
	st, err := store.Open(store.DefaultDBPath())
	if err != nil {
		return fmt.Errorf("opening store: %w", err)
	}
	m.store = st

	cm := proxy.NewCertManager(configDir)
	ca, err := cm.EnsureCA()
	if err != nil {
		return fmt.Errorf("loading CA: %w", err)
	}

	p, err := proxy.New(ca, st, m.sb.ID, m.sb.Config.Secrets)
	if err != nil {
		return fmt.Errorf("creating proxy: %w", err)
	}
	m.proxy = p

	proxyAddr, err := p.Start(m.sb.Config.ProxyPort)
	if err != nil {
		return fmt.Errorf("starting proxy: %w", err)
	}
	// Extract port from the listen address for use with host.docker.internal.
	_, port, _ := net.SplitHostPort(proxyAddr)
	m.sb.ProxyAddr = "host.docker.internal:" + port

	// 2. Create network.
	netName := fmt.Sprintf("codingbox-%s", m.sb.ID)
	netID, err := CreateNetwork(ctx, m.cli, netName)
	if err != nil {
		return err
	}
	m.sb.NetworkID = netID

	// 3. Build mounts.
	mounts := []mount.Mount{
		{
			Type:   mount.TypeBind,
			Source: m.sb.Config.Workdir,
			Target: "/workspace",
		},
	}
	// Mount CA cert into container for TLS interception.
	mounts = append(mounts, mount.Mount{
		Type:     mount.TypeBind,
		Source:   cm.CertPath(),
		Target:   "/usr/local/share/ca-certificates/codingbox-ca.crt",
		ReadOnly: true,
	})
	for _, mc := range m.sb.Config.Mounts {
		mounts = append(mounts, mount.Mount{
			Type:     mount.TypeBind,
			Source:   mc.Source,
			Target:   mc.Target,
			ReadOnly: mc.Mode == "ro",
		})
	}

	env := m.buildEnv()

	// 3. Create container.
	containerName := fmt.Sprintf("codingbox-%s", m.sb.ID)
	createResp, err := m.cli.ContainerCreate(ctx, client.ContainerCreateOptions{
		Config: &container.Config{
			Image:        m.sb.Config.Image,
			Tty:          true,
			OpenStdin:    true,
			AttachStdin:  true,
			AttachStdout: true,
			AttachStderr: true,
			WorkingDir:   "/workspace",
			Env:          env,
		},
		HostConfig: &container.HostConfig{
			Mounts:     mounts,
			ExtraHosts: []string{"host.docker.internal:host-gateway"},
		},
		NetworkingConfig: &network.NetworkingConfig{
			EndpointsConfig: map[string]*network.EndpointSettings{
				netName: {},
			},
		},
		Name: containerName,
	})
	if err != nil {
		return fmt.Errorf("creating container: %w", err)
	}
	m.sb.ContainerID = createResp.ID

	// 4. Start container.
	if _, err := m.cli.ContainerStart(ctx, m.sb.ContainerID, client.ContainerStartOptions{}); err != nil {
		return fmt.Errorf("starting container: %w", err)
	}
	m.sb.State = models.StateRunning

	// 5. Attach interactive TTY.
	if err := AttachInteractive(ctx, m.cli, m.sb.ContainerID); err != nil {
		if ctx.Err() != nil {
			return nil
		}
		return fmt.Errorf("attaching to container: %w", err)
	}

	return nil
}

// Stop cleans up all sandbox resources.
func (m *Manager) Stop() {
	m.sb.State = models.StateStopping

	cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cleanupCancel()

	if m.sb.ContainerID != "" {
		timeout := 3
		_, _ = m.cli.ContainerStop(cleanupCtx, m.sb.ContainerID, client.ContainerStopOptions{Timeout: &timeout})
		_, _ = m.cli.ContainerRemove(cleanupCtx, m.sb.ContainerID, client.ContainerRemoveOptions{Force: true})
		m.sb.ContainerID = ""
	}

	if m.sb.NetworkID != "" {
		_ = RemoveNetwork(cleanupCtx, m.cli, m.sb.NetworkID)
		m.sb.NetworkID = ""
	}

	if m.proxy != nil {
		m.proxy.Stop()
	}
	if m.store != nil {
		m.store.Close()
	}

	m.sb.State = models.StateStopped
	m.cli.Close()
}

// buildEnv returns environment variables for the container.
func (m *Manager) buildEnv() []string {
	var env []string
	if m.sb.ProxyAddr != "" {
		// The proxy listens on the host. From the container, use host.docker.internal
		// or the gateway IP. We'll use the host's IP on the Docker bridge.
		proxyURL := fmt.Sprintf("http://%s", m.sb.ProxyAddr)
		env = append(env,
			"HTTP_PROXY="+proxyURL,
			"HTTPS_PROXY="+proxyURL,
			"http_proxy="+proxyURL,
			"https_proxy="+proxyURL,
			"SSL_CERT_FILE=/usr/local/share/ca-certificates/codingbox-ca.crt",
			"NODE_EXTRA_CA_CERTS=/usr/local/share/ca-certificates/codingbox-ca.crt",
		)
	}
	return env
}

// Sandbox returns the current sandbox state.
func (m *Manager) Sandbox() *models.Sandbox {
	return m.sb
}

// DockerClient returns the Docker client.
func (m *Manager) DockerClient() *client.Client {
	return m.cli
}
