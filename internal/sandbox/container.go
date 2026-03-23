package sandbox

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"strings"

	"github.com/codingbox/codingbox/internal/models"
)

// ContainerConfig holds configuration for launching an agent container inside a microVM.
type ContainerConfig struct {
	BaseImage    string
	WorkspaceDir string
	Mounts       []models.Mount
	Secrets      []models.SecretMapping
	ProxyPort    int
	CACertPEM    []byte
	Tools        []string
	SessionID    string
	StateDir     string // Host-side directory for persistent state
}

// ContainerManager manages containers inside a microVM's Docker daemon.
type ContainerManager struct {
	vmSocketPath string
	containerID  string
	logger       *slog.Logger
}

// NewContainerManager creates a new container manager.
func NewContainerManager(vmSocketPath string, logger *slog.Logger) *ContainerManager {
	return &ContainerManager{
		vmSocketPath: vmSocketPath,
		logger:       logger,
	}
}

// Start launches the agent container inside the microVM.
func (cm *ContainerManager) Start(ctx context.Context, cfg ContainerConfig) error {
	baseImage := cfg.BaseImage
	if baseImage == "" {
		baseImage = "ubuntu:22.04"
	}

	// Load image into the VM's Docker daemon
	cm.logger.Info("loading base image into microVM", "image", baseImage)
	if err := cm.loadImage(ctx, baseImage); err != nil {
		return fmt.Errorf("loading image: %w", err)
	}

	// Build docker run arguments
	args := []string{
		"--host", "unix://" + cm.vmSocketPath,
		"run", "-d",
		"--name", "codingbox-agent-" + cfg.SessionID[:8],
	}

	// Proxy environment variables
	proxyURL := fmt.Sprintf("http://host.docker.internal:%d", cfg.ProxyPort)
	args = append(args, "-e", "HTTP_PROXY="+proxyURL)
	args = append(args, "-e", "HTTPS_PROXY="+proxyURL)
	args = append(args, "-e", "http_proxy="+proxyURL)
	args = append(args, "-e", "https_proxy="+proxyURL)

	// Workspace mount
	args = append(args, "-v", cfg.WorkspaceDir+":/workspace:rw")
	args = append(args, "-w", "/workspace")

	// Additional mounts
	for _, m := range cfg.Mounts {
		args = append(args, "-v", fmt.Sprintf("%s:%s:%s", m.HostPath, m.SandboxPath, m.Mode))
	}

	// Secret placeholder environment variables (agent sees only UUIDs)
	for _, s := range cfg.Secrets {
		envName := strings.ToUpper(strings.ReplaceAll(s.Name, "-", "_")) + "_PLACEHOLDER"
		args = append(args, "-e", envName+"="+s.ID)
	}

	// Persistent state directory on host (survives VM destroy/recreate)
	if cfg.StateDir != "" {
		args = append(args, "-v", cfg.StateDir+":/home/agent/.local:rw")
	}

	// CA certificate mount for HTTPS interception
	if len(cfg.CACertPEM) > 0 {
		tmpCert, err := writeTempCACert(cfg.CACertPEM)
		if err != nil {
			cm.logger.Warn("failed to write CA cert for container", "error", err)
		} else {
			args = append(args, "-v", tmpCert+":/usr/local/share/ca-certificates/codingbox-ca.crt:ro")
		}
	}

	args = append(args, baseImage)
	args = append(args, "sleep", "infinity") // Keep container alive

	cm.logger.Debug("starting container", "args", args)

	cmd := exec.CommandContext(ctx, "docker", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("starting container: %w: %s", err, string(output))
	}

	cm.containerID = strings.TrimSpace(string(output))
	cm.logger.Info("container started", "container_id", cm.containerID[:12])

	return nil
}

// Stop stops the agent container.
func (cm *ContainerManager) Stop(ctx context.Context, force bool) error {
	if cm.containerID == "" {
		return nil
	}

	args := []string{"--host", "unix://" + cm.vmSocketPath}
	if force {
		args = append(args, "kill", cm.containerID)
	} else {
		args = append(args, "stop", cm.containerID)
	}

	cmd := exec.CommandContext(ctx, "docker", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("stopping container: %w: %s", err, string(output))
	}

	// Remove container
	rmArgs := []string{"--host", "unix://" + cm.vmSocketPath, "rm", "-f", cm.containerID}
	rmCmd := exec.CommandContext(ctx, "docker", rmArgs...)
	rmCmd.CombinedOutput() // best-effort cleanup

	return nil
}

// Exec starts an interactive shell inside the running container.
func (cm *ContainerManager) Exec(ctx context.Context) error {
	if cm.containerID == "" {
		return fmt.Errorf("no container running")
	}

	args := []string{
		"--host", "unix://" + cm.vmSocketPath,
		"exec", "-it", cm.containerID,
		"/bin/sh", "-c", "if command -v bash >/dev/null 2>&1; then exec bash; else exec sh; fi",
	}

	cmd := exec.CommandContext(ctx, "docker", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// ContainerID returns the running container's ID.
func (cm *ContainerManager) ContainerID() string {
	return cm.containerID
}

func (cm *ContainerManager) loadImage(ctx context.Context, image string) error {
	// Try pulling on host first, then save/load into VM
	pullCmd := exec.CommandContext(ctx, "docker", "pull", image)
	if output, err := pullCmd.CombinedOutput(); err != nil {
		cm.logger.Debug("pull failed, image may already exist", "error", err, "output", string(output))
	}

	// Pipe: docker save | docker --host unix://VM_SOCK load
	saveCmd := exec.CommandContext(ctx, "docker", "save", image)
	loadCmd := exec.CommandContext(ctx, "docker", "--host", "unix://"+cm.vmSocketPath, "load")

	var saveBuf, loadBuf bytes.Buffer
	saveCmd.Stderr = &saveBuf
	loadCmd.Stderr = &loadBuf

	pr, pw := io.Pipe()
	saveCmd.Stdout = pw
	loadCmd.Stdin = pr

	if err := saveCmd.Start(); err != nil {
		return fmt.Errorf("starting docker save: %w", err)
	}
	if err := loadCmd.Start(); err != nil {
		return fmt.Errorf("starting docker load: %w", err)
	}

	saveErr := make(chan error, 1)
	go func() {
		saveErr <- saveCmd.Wait()
		pw.Close()
	}()

	if err := loadCmd.Wait(); err != nil {
		return fmt.Errorf("docker load into VM: %w\n  load stderr: %s\n  save stderr: %s",
			err, strings.TrimSpace(loadBuf.String()), strings.TrimSpace(saveBuf.String()))
	}

	if err := <-saveErr; err != nil {
		return fmt.Errorf("docker save: %w\n  stderr: %s", err, strings.TrimSpace(saveBuf.String()))
	}

	return nil
}

func writeTempCACert(certPEM []byte) (string, error) {
	tmpFile, err := os.CreateTemp("", "codingbox-ca-*.crt")
	if err != nil {
		return "", err
	}
	defer tmpFile.Close()

	if _, err := tmpFile.Write(certPEM); err != nil {
		os.Remove(tmpFile.Name())
		return "", err
	}

	return tmpFile.Name(), nil
}
