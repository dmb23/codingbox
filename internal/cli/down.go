package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/codingbox/codingbox/internal/config"
	"github.com/codingbox/codingbox/internal/models"
	"github.com/codingbox/codingbox/internal/sandbox"
	"github.com/codingbox/codingbox/internal/store"
	"github.com/spf13/cobra"
)

var forceDown bool

var downCmd = &cobra.Command{
	Use:   "down <session-id | session-name>",
	Short: "Stop a running sandbox session",
	Args:  cobra.ExactArgs(1),
	RunE:  runDown,
}

func init() {
	downCmd.Flags().BoolVar(&forceDown, "force", false, "kill immediately without graceful shutdown")
	rootCmd.AddCommand(downCmd)
}

func runDown(cmd *cobra.Command, args []string) error {
	identifier := args[0]

	globalCfg, err := config.LoadGlobalConfig()
	if err != nil {
		return fmt.Errorf("codingbox: error: %w", err)
	}

	db, err := store.Open(globalCfg.DBPath)
	if err != nil {
		return fmt.Errorf("codingbox: error: %w", err)
	}
	defer db.Close()

	sessionStore := store.NewSessionStore(db)

	// Find session by ID or name
	session, err := sessionStore.Get(identifier)
	if err != nil {
		session, err = sessionStore.FindByName(identifier)
		if err != nil {
			return fmt.Errorf("codingbox: error: session not found: %s", identifier)
		}
	}

	if session.Status != models.StatusRunning && session.Status != models.StatusCreated {
		return fmt.Errorf("codingbox: error: session %s is already %s", session.ID, session.Status)
	}

	// Destroy VM
	vmClient := sandbox.NewClient("")
	var cfg models.SandboxConfig
	if err := json.Unmarshal([]byte(session.ConfigSnapshot), &cfg); err == nil {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := vmClient.DestroyVM(ctx, cfg.Name); err != nil {
			logger.Warn("failed to destroy VM", "error", err)
		}
	}

	// Update session status
	if err := sessionStore.UpdateStatus(session.ID, models.StatusStopped, ""); err != nil {
		return fmt.Errorf("codingbox: error: %w", err)
	}

	fmt.Printf("Session %s stopped\n", session.ID)
	return nil
}
