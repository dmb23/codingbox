package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/codingbox/codingbox/internal/config"
	"github.com/codingbox/codingbox/internal/proxy"
	"github.com/codingbox/codingbox/internal/sandbox"
	"github.com/codingbox/codingbox/internal/store"
	"github.com/spf13/cobra"
)

var (
	sessionName string
	detach      bool
)

var upCmd = &cobra.Command{
	Use:   "up",
	Short: "Start a sandbox session",
	Long:  "Start a sandbox session from a codingbox.yml configuration file.",
	RunE:  runUp,
}

func init() {
	upCmd.Flags().StringVar(&sessionName, "name", "", "human-readable session name")
	upCmd.Flags().BoolVar(&detach, "detach", false, "run in background, print session ID")
	rootCmd.AddCommand(upCmd)
}

func runUp(cmd *cobra.Command, args []string) error {
	// Load sandbox config
	cfg, err := config.LoadSandboxConfig(cfgFile)
	if err != nil {
		return fmt.Errorf("codingbox: error: %w", err)
	}

	if sessionName != "" {
		cfg.Name = sessionName
	}

	// Load global config
	globalCfg, err := config.LoadGlobalConfig()
	if err != nil {
		return fmt.Errorf("codingbox: error: %w", err)
	}

	// Initialize CA
	ca, err := proxy.LoadOrGenerateCA(globalCfg.CACertPath, globalCfg.CAKeyPath)
	if err != nil {
		return fmt.Errorf("codingbox: error: %w", err)
	}

	// Initialize database
	db, err := store.Open(globalCfg.DBPath)
	if err != nil {
		return fmt.Errorf("codingbox: error: %w", err)
	}
	defer db.Close()

	sessionStore := store.NewSessionStore(db)
	logStore := store.NewLogStore(db)

	// Create orchestrator
	vmClient := sandbox.NewClient("")
	orchestrator := sandbox.NewSessionOrchestrator(vmClient, sessionStore, logStore, ca, logger)

	// Run log retention on session creation
	orchestrator.RunLogRetention(globalCfg.LogRetentionDays)

	// Handle Ctrl+C
	ctx, cancel := context.WithCancel(cmd.Context())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		logger.Info("received shutdown signal")
		orchestrator.Stop(context.Background(), false)
		cancel()
	}()

	// Start session
	session, err := orchestrator.Start(ctx, cfg)
	if err != nil {
		return fmt.Errorf("codingbox: error: %w", err)
	}

	if detach {
		fmt.Println(session.ID)
		return nil
	}

	// Foreground mode: stream container logs
	if err := orchestrator.StreamLogs(ctx); err != nil {
		logger.Debug("stream ended", "error", err)
	}

	// Graceful shutdown on stream end
	orchestrator.Stop(context.Background(), false)

	return nil
}
