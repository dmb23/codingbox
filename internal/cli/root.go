package cli

import (
	"log/slog"
	"os"

	"github.com/spf13/cobra"
)

var (
	cfgFile string
	verbose bool
	logger  *slog.Logger
)

var rootCmd = &cobra.Command{
	Use:   "codingbox",
	Short: "Secure sandbox environments for coding agents",
	Long: `codingbox creates isolated microVM sandbox environments for coding agents.
All outbound HTTP traffic is routed through a MITM proxy that injects secrets
per-host and logs every request to SQLite for full observability.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		level := slog.LevelInfo
		if verbose {
			level = slog.LevelDebug
		}
		logger = slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
			Level: level,
		}))
	},
	SilenceUsage:  true,
	SilenceErrors: true,
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "./codingbox.yml", "path to sandbox configuration file")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "enable verbose (debug) logging")
}

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}

// Version is set at build time.
var Version = "dev"
