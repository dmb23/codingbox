package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/codingbox/codingbox/internal/config"
	"github.com/codingbox/codingbox/internal/models"
	"github.com/codingbox/codingbox/internal/store"
	"github.com/spf13/cobra"
)

var (
	psAll    bool
	psFormat string
)

var psCmd = &cobra.Command{
	Use:   "ps",
	Short: "List sandbox sessions",
	RunE:  runPS,
}

func init() {
	psCmd.Flags().BoolVar(&psAll, "all", false, "include stopped sessions")
	psCmd.Flags().StringVar(&psFormat, "format", "table", "output format (table|json)")
	rootCmd.AddCommand(psCmd)
}

func runPS(cmd *cobra.Command, args []string) error {
	globalCfg, err := config.LoadGlobalConfig()
	if err != nil {
		return fmt.Errorf("%w", err)
	}

	db, err := store.Open(globalCfg.DBPath)
	if err != nil {
		return fmt.Errorf("%w", err)
	}
	defer db.Close()

	sessionStore := store.NewSessionStore(db)

	var statusFilter string
	if !psAll {
		statusFilter = models.StatusRunning
	}

	sessions, err := sessionStore.List(statusFilter)
	if err != nil {
		return fmt.Errorf("%w", err)
	}

	if psFormat == "json" {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(sessions)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, "SESSION ID\tNAME\tSTATUS\tAGENT\tCREATED")

	for _, s := range sessions {
		name := extractName(s.ConfigSnapshot)
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			s.ID, name, s.Status, s.AgentName,
			s.CreatedAt.Format("2006-01-02T15:04:05Z"))
	}

	return w.Flush()
}

func extractName(configSnapshot string) string {
	var cfg struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal([]byte(configSnapshot), &cfg); err == nil && cfg.Name != "" {
		return cfg.Name
	}
	return "-"
}
