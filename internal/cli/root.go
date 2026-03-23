package cli

import "github.com/spf13/cobra"

var rootCmd = &cobra.Command{
	Use:   "codingbox",
	Short: "Secure sandbox for agentic coding workloads",
	Long:  "codingbox launches OCI containers as sandboxed environments for coding agents with traffic logging and secret injection.",
}

func Execute() error {
	return rootCmd.Execute()
}
