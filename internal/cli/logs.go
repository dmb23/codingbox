package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/mischa/codingbox/internal/store"
	"github.com/spf13/cobra"
)

var logsCmd = &cobra.Command{
	Use:   "logs",
	Short: "Query traffic logs",
	Long:  "Query traffic logs from a previous or current sandbox session.",
	RunE:  runLogs,
}

func init() {
	logsCmd.Flags().String("session", "", "Session ID to query (default: most recent)")
	logsCmd.Flags().String("method", "", "Filter by HTTP method")
	logsCmd.Flags().String("url", "", "Filter by URL pattern (substring match)")
	logsCmd.Flags().Int("status", 0, "Filter by response status code")
	logsCmd.Flags().String("since", "", "Show logs since timestamp (RFC3339)")
	logsCmd.Flags().IntP("limit", "n", 50, "Maximum number of entries")
	logsCmd.Flags().StringP("format", "f", "table", "Output format: table, json")
	logsCmd.Flags().Bool("body", false, "Include request/response bodies")

	rootCmd.AddCommand(logsCmd)
}

func runLogs(cmd *cobra.Command, args []string) error {
	session, _ := cmd.Flags().GetString("session")
	method, _ := cmd.Flags().GetString("method")
	url, _ := cmd.Flags().GetString("url")
	status, _ := cmd.Flags().GetInt("status")
	sinceStr, _ := cmd.Flags().GetString("since")
	limit, _ := cmd.Flags().GetInt("limit")
	format, _ := cmd.Flags().GetString("format")
	showBody, _ := cmd.Flags().GetBool("body")

	st, err := store.Open(store.DefaultDBPath())
	if err != nil {
		return fmt.Errorf("opening store: %w", err)
	}
	defer st.Close()

	if session == "" {
		session, _ = st.LatestSandboxID()
	}

	var since time.Time
	if sinceStr != "" {
		since, err = time.Parse(time.RFC3339, sinceStr)
		if err != nil {
			return fmt.Errorf("invalid --since format (expected RFC3339): %w", err)
		}
	}

	logs, err := st.QueryLogs(store.LogFilter{
		SandboxID: session,
		Method:    method,
		URL:       url,
		Status:    status,
		Since:     since,
		Limit:     limit,
	})
	if err != nil {
		return fmt.Errorf("querying logs: %w", err)
	}

	if format == "json" {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		for _, l := range logs {
			if !showBody {
				l.RequestBody = nil
				l.ResponseBody = nil
			}
			enc.Encode(l)
		}
		return nil
	}

	// Table format.
	w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, "TIME\tMETHOD\tURL\tSTATUS\tDURATION\tSECRETS")
	for _, l := range logs {
		fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%dms\t%v\n",
			l.Timestamp.Format("15:04:05"),
			l.Method,
			truncate(l.URL, 60),
			l.ResponseStatus,
			l.DurationMs,
			l.SecretsReplaced,
		)
		if showBody {
			if len(l.RequestBody) > 0 {
				fmt.Fprintf(w, "  REQ BODY:\t%s\n", truncate(string(l.RequestBody), 200))
			}
			if len(l.ResponseBody) > 0 {
				fmt.Fprintf(w, "  RESP BODY:\t%s\n", truncate(string(l.ResponseBody), 200))
			}
		}
	}
	w.Flush()
	return nil
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
