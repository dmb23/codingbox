package sandbox

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"syscall"
	"time"

	"github.com/codingbox/codingbox/internal/models"
	"github.com/codingbox/codingbox/internal/proxy"
	"github.com/codingbox/codingbox/internal/store"
	"github.com/oklog/ulid/v2"
)

const minFreeSpaceMB = 100

// SessionOrchestrator coordinates VM creation, proxy startup, container launch, and teardown.
type SessionOrchestrator struct {
	vmClient     *Client
	sessionStore *store.SessionStore
	logStore     *store.LogStore
	ca           *proxy.CA
	logger       *slog.Logger

	// Active session state
	session      *models.SandboxSession
	proxyServer  *proxy.Server
	container    *ContainerManager
	interceptor  *proxy.Interceptor
}

// NewSessionOrchestrator creates a new orchestrator.
func NewSessionOrchestrator(
	vmClient *Client,
	sessionStore *store.SessionStore,
	logStore *store.LogStore,
	ca *proxy.CA,
	logger *slog.Logger,
) *SessionOrchestrator {
	return &SessionOrchestrator{
		vmClient:     vmClient,
		sessionStore: sessionStore,
		logStore:     logStore,
		ca:           ca,
		logger:       logger,
	}
}

// Start launches a new sandbox session from the given config.
func (o *SessionOrchestrator) Start(ctx context.Context, cfg *models.SandboxConfig) (*models.SandboxSession, error) {
	sessionID := ulid.Make().String()

	configJSON, err := json.Marshal(cfg)
	if err != nil {
		return nil, fmt.Errorf("serializing config: %w", err)
	}

	session := &models.SandboxSession{
		ID:             sessionID,
		AgentName:      cfg.Agent,
		Status:         models.StatusCreated,
		ConfigSnapshot: string(configJSON),
		CreatedAt:      time.Now(),
	}

	// 0. Check disk space
	if err := checkDiskSpace(cfg.WorkspaceDir, o.logger); err != nil {
		return nil, err
	}

	// 1. Create microVM
	o.logger.Info("creating microVM", "session_id", sessionID, "agent", cfg.Agent)
	vm, err := o.vmClient.CreateVM(ctx, VMCreateRequest{
		AgentName:    cfg.Agent,
		WorkspaceDir: cfg.WorkspaceDir,
	})
	if err != nil {
		session.Status = models.StatusFailed
		session.ErrorMessage = fmt.Sprintf("VM creation failed: %v", err)
		session.VMID = ""
		session.VMSocketPath = ""
		_ = o.sessionStore.Create(session)
		return nil, fmt.Errorf("creating VM: %w", err)
	}

	session.VMID = vm.VMID
	session.VMSocketPath = vm.SocketPath

	// 2. Persist session
	if err := o.sessionStore.Create(session); err != nil {
		_ = o.vmClient.DestroyVM(ctx, vm.Name)
		return nil, fmt.Errorf("persisting session: %w", err)
	}

	// 3. Start proxy
	proxyServer, err := proxy.NewServer(o.ca, cfg.Proxy.Port, o.logger)
	if err != nil {
		o.failSession(ctx, session, vm.Name, fmt.Sprintf("proxy setup failed: %v", err))
		return nil, fmt.Errorf("creating proxy: %w", err)
	}

	interceptor := proxy.NewInterceptor(sessionID)

	// Wire up secret injection
	if len(cfg.Secrets) > 0 {
		injector := proxy.NewSecretInjector(cfg.Secrets)
		interceptor.SetInjector(injector)

		// Wire up logging with secret redaction
		reqLogger := proxy.NewRequestLogger(o.logStore, injector)
		interceptor.SetLogger(reqLogger)
	} else {
		reqLogger := proxy.NewRequestLogger(o.logStore, nil)
		interceptor.SetLogger(reqLogger)
	}

	interceptor.Install(proxyServer.Proxy())

	proxyPort, err := proxyServer.Start()
	if err != nil {
		o.failSession(ctx, session, vm.Name, fmt.Sprintf("proxy start failed: %v", err))
		return nil, fmt.Errorf("starting proxy: %w", err)
	}

	o.proxyServer = proxyServer
	o.interceptor = interceptor
	o.logger.Info("proxy started", "port", proxyPort)

	// 4. Start container inside microVM
	containerMgr := NewContainerManager(vm.SocketPath, o.logger)
	o.container = containerMgr

	containerCfg := ContainerConfig{
		BaseImage:    cfg.BaseImage,
		WorkspaceDir: cfg.WorkspaceDir,
		Mounts:       cfg.Mounts,
		Secrets:      cfg.Secrets,
		ProxyPort:    proxyPort,
		CACertPEM:    o.ca.CertPEM,
		Tools:        cfg.Tools,
		SessionID:    sessionID,
	}

	if err := containerMgr.Start(ctx, containerCfg); err != nil {
		proxyServer.Shutdown(ctx)
		o.failSession(ctx, session, vm.Name, fmt.Sprintf("container start failed: %v", err))
		return nil, fmt.Errorf("starting container: %w", err)
	}

	// 5. Transition to running
	if err := o.sessionStore.UpdateStatus(sessionID, models.StatusRunning, ""); err != nil {
		o.logger.Error("failed to update session status", "error", err)
	}

	o.session = session
	o.logger.Info("sandbox session started", "session_id", sessionID, "vm_id", vm.VMID)

	return session, nil
}

// Stop gracefully shuts down a sandbox session.
func (o *SessionOrchestrator) Stop(ctx context.Context, force bool) error {
	if o.session == nil {
		return fmt.Errorf("no active session")
	}

	o.logger.Info("stopping sandbox session", "session_id", o.session.ID)

	// Stop container
	if o.container != nil {
		if err := o.container.Stop(ctx, force); err != nil {
			o.logger.Error("failed to stop container", "error", err)
		}
	}

	// Stop proxy
	if o.proxyServer != nil {
		if err := o.proxyServer.Shutdown(ctx); err != nil {
			o.logger.Error("failed to stop proxy", "error", err)
		}
	}

	// Destroy VM
	if o.session.VMID != "" {
		// Use the config snapshot to get the VM name
		var cfg models.SandboxConfig
		if err := json.Unmarshal([]byte(o.session.ConfigSnapshot), &cfg); err == nil {
			if err := o.vmClient.DestroyVM(ctx, cfg.Name); err != nil {
				o.logger.Error("failed to destroy VM", "error", err)
			}
		}
	}

	// Update session status
	if err := o.sessionStore.UpdateStatus(o.session.ID, models.StatusStopped, ""); err != nil {
		o.logger.Error("failed to update session status", "error", err)
	}

	return nil
}

// StreamLogs streams container output to the given writer.
func (o *SessionOrchestrator) StreamLogs(ctx context.Context) error {
	if o.container == nil {
		return fmt.Errorf("no active container")
	}
	return o.container.StreamLogs(ctx)
}

// Session returns the current session.
func (o *SessionOrchestrator) Session() *models.SandboxSession {
	return o.session
}

func (o *SessionOrchestrator) failSession(ctx context.Context, session *models.SandboxSession, vmName, errMsg string) {
	_ = o.sessionStore.UpdateStatus(session.ID, models.StatusFailed, errMsg)
	if vmName != "" {
		_ = o.vmClient.DestroyVM(ctx, vmName)
	}
}

// checkDiskSpace verifies sufficient free space is available.
func checkDiskSpace(path string, logger *slog.Logger) error {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		logger.Warn("could not check disk space", "path", path, "error", err)
		return nil // non-fatal
	}
	freeMB := (stat.Bavail * uint64(stat.Bsize)) / (1024 * 1024)
	if freeMB < minFreeSpaceMB {
		return fmt.Errorf("codingbox: error: low disk space on %s (%d MB free, need at least %d MB). "+
			"Free up space by removing old sessions or running: codingbox logs --cleanup", path, freeMB, minFreeSpaceMB)
	}
	return nil
}

// RunLogRetention deletes old log entries based on retention config.
func (o *SessionOrchestrator) RunLogRetention(retentionDays int) {
	if retentionDays <= 0 {
		return
	}
	deleted, err := o.logStore.DeleteOlderThan(retentionDays)
	if err != nil {
		o.logger.Warn("log retention cleanup failed", "error", err)
	} else if deleted > 0 {
		o.logger.Info("log retention cleanup", "deleted", deleted, "retention_days", retentionDays)
	}
}
