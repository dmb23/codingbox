package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/codingbox/codingbox/internal/config"
	"github.com/codingbox/codingbox/internal/store"
	"github.com/spf13/cobra"
)

var (
	logsHost   string
	logsStatus int
	logsSince  string
	logsUntil  string
	logsFormat string
	logsLimit  int
)

var logsCmd = &cobra.Command{
	Use:   "logs <session-id>",
	Short: "Query request logs for a session",
	Args:  cobra.ExactArgs(1),
	RunE:  runLogs,
}

func init() {
	logsCmd.Flags().StringVar(&logsHost, "host", "", "filter by target hostname")
	logsCmd.Flags().IntVar(&logsStatus, "status", 0, "filter by HTTP status code")
	logsCmd.Flags().StringVar(&logsSince, "since", "", "only entries after this timestamp")
	logsCmd.Flags().StringVar(&logsUntil, "until", "", "only entries before this timestamp")
	logsCmd.Flags().StringVar(&logsFormat, "format", "table", "output format (table|json)")
	logsCmd.Flags().IntVar(&logsLimit, "limit", 100, "maximum entries to return")
	rootCmd.AddCommand(logsCmd)
}

func runLogs(cmd *cobra.Command, args []string) error {
	sessionID := args[0]

	globalCfg, err := config.LoadGlobalConfig()
	if err != nil {
		return fmt.Errorf("codingbox: error: %w", err)
	}

	db, err := store.Open(globalCfg.DBPath)
	if err != nil {
		return fmt.Errorf("codingbox: error: %w", err)
	}
	defer db.Close()

	logStore := store.NewLogStore(db)

	q := store.LogQuery{
		SessionID: sessionID,
		Host:      logsHost,
		Limit:     logsLimit,
	}

	if logsStatus != 0 {
		q.Status = &logsStatus
	}

	if logsSince != "" {
		t, err := time.Parse(time.RFC3339, logsSince)
		if err != nil {
			return fmt.Errorf("codingbox: error: invalid --since timestamp: %w", err)
		}
		q.Since = &t
	}

	if logsUntil != "" {
		t, err := time.Parse(time.RFC3339, logsUntil)
		if err != nil {
			return fmt.Errorf("codingbox: error: invalid --until timestamp: %w", err)
		}
		q.Until = &t
	}

	entries, err := logStore.Query(q)
	if err != nil {
		return fmt.Errorf("codingbox: error: %w", err)
	}

	if logsFormat == "json" {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(entries)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, "TIMESTAMP\tMETHOD\tURL\tSTATUS\tLATENCY\tSECRETS")

	for _, e := range entries {
		status := "-"
		if e.ResponseStatus != nil {
			status = fmt.Sprintf("%d", *e.ResponseStatus)
		}
		latency := "-"
		if e.LatencyMs != nil {
			latency = fmt.Sprintf("%dms", *e.LatencyMs)
		}
		secrets := "[]"
		if e.SecretsInjected != "" {
			secrets = e.SecretsInjected
		}

		url := e.URL
		if len(url) > 50 {
			url = url[:47] + "..."
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
			e.Timestamp.Format("2006-01-02T15:04:05Z"),
			e.Method, url, status, latency, secrets)
	}

	return w.Flush()
}
