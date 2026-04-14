package sandbox

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/mount"
	"github.com/moby/moby/api/types/network"
	"github.com/moby/moby/client"

	"github.com/mischa/codingbox/internal/config"
	"github.com/mischa/codingbox/internal/models"
	"github.com/mischa/codingbox/internal/proxy"
	"github.com/mischa/codingbox/internal/store"
)

// Manager handles the sandbox lifecycle.
type Manager struct {
	cli      *client.Client
	sb       *models.Sandbox
	proxy    *proxy.Proxy
	store    *store.Store
	cancel   context.CancelFunc
	stopOnce sync.Once
}

// NewManager creates a new sandbox manager.
func NewManager(cfg *models.SandboxConfig) (*Manager, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("connecting to Docker: %w\nIs Docker installed? Check with: docker --version", err)
	}

	// Verify Docker daemon is reachable.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := cli.Ping(ctx, client.PingOptions{}); err != nil {
		cli.Close()
		return nil, fmt.Errorf("Docker daemon is not running: %w\nStart Docker with: docker info", err)
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
		if strings.Contains(err.Error(), "address already in use") {
			return fmt.Errorf("proxy port %d is already in use: %w\nTry a different port with --proxy-port or use 0 for auto-assign", m.sb.Config.ProxyPort, err)
		}
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

	// Auto-mounts: add config directories from host (same path inside container).
	if !m.sb.Config.NoAutoMounts {
		explicitTargets := make(map[string]bool)
		for _, mc := range m.sb.Config.Mounts {
			explicitTargets[mc.Target] = true
		}
		autoMounts := config.ResolveAutoMounts(os.Getenv("HOME"))
		for _, am := range autoMounts {
			if explicitTargets[am.Target] {
				continue // explicit mount takes precedence
			}
			mounts = append(mounts, mount.Mount{
				Type:     mount.TypeBind,
				Source:   am.Source,
				Target:   am.Target,
				ReadOnly: am.Mode == "ro",
			})
		}
	}

	for _, mc := range m.sb.Config.Mounts {
		mounts = append(mounts, mount.Mount{
			Type:     mount.TypeBind,
			Source:   mc.Source,
			Target:   mc.Target,
			ReadOnly: mc.Mode == "ro",
		})
	}

	// 3a. Ensure image is available locally (pull if needed).
	if err := m.ensureImage(ctx); err != nil {
		return err
	}

	env := m.buildEnv()

	// 3b. Create container.
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
		if strings.Contains(err.Error(), "permission denied") {
			return fmt.Errorf("creating container: %w\nCheck mount directory permissions. Ensure Docker has access to the directories being mounted", err)
		}
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

// Stop cleans up all sandbox resources. It is safe to call multiple times.
func (m *Manager) Stop() {
	m.stopOnce.Do(func() {
		m.sb.State = models.StateStopping

		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cleanupCancel()

		if m.sb.ContainerID != "" {
			timeout := 3
			if _, err := m.cli.ContainerStop(cleanupCtx, m.sb.ContainerID, client.ContainerStopOptions{Timeout: &timeout}); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to stop container: %v\n", err)
			}
			if _, err := m.cli.ContainerRemove(cleanupCtx, m.sb.ContainerID, client.ContainerRemoveOptions{Force: true}); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to remove container: %v\n", err)
			}
			m.sb.ContainerID = ""
		}

		if m.sb.NetworkID != "" {
			if err := RemoveNetwork(cleanupCtx, m.cli, m.sb.NetworkID); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to remove network: %v\n", err)
			}
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
	})
}

// ensureImage checks if the image exists locally and pulls it if not.
func (m *Manager) ensureImage(ctx context.Context) error {
	img := m.sb.Config.Image

	// Check if image exists locally.
	_, err := m.cli.ImageInspect(ctx, img)
	if err == nil {
		return nil // Image exists locally.
	}

	// Image not found locally — attempt to pull.
	fmt.Fprintf(os.Stderr, "Image %q not found locally, pulling...\n", img)
	pullResp, err := m.cli.ImagePull(ctx, img, client.ImagePullOptions{})
	if err != nil {
		return fmt.Errorf("pulling image %q: %w\nEnsure the image name is correct and accessible. For local images, build with: docker build -t %s .", img, err, img)
	}
	defer pullResp.Close()

	// Consume the pull output (required to complete the pull).
	if _, err := io.Copy(os.Stderr, pullResp); err != nil {
		return fmt.Errorf("reading pull progress: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Successfully pulled %q\n", img)
	return nil
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

	// Inject env-based secrets as environment variables (placeholder values).
	for _, s := range m.sb.Config.Secrets {
		if s.Env != "" {
			env = append(env, fmt.Sprintf("%s=%s", s.Env, s.Placeholder))
		}
	}

	// Pass host UID/GID/HOME for entrypoint user matching.
	env = append(env,
		fmt.Sprintf("CODINGBOX_UID=%d", os.Getuid()),
		fmt.Sprintf("CODINGBOX_GID=%d", os.Getgid()),
		fmt.Sprintf("CODINGBOX_HOME=%s", os.Getenv("HOME")),
	)

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
