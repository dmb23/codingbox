package cli

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/mischa/codingbox/internal/config"
	"github.com/mischa/codingbox/internal/sandbox"
	"github.com/spf13/cobra"
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Launch a sandbox session",
	Long:  "Launch an interactive sandbox session from a configuration file or CLI flags.",
	RunE:  runRun,
}

func init() {
	runCmd.Flags().StringP("config", "c", "", "Path to configuration file (default: ./codingbox.yaml)")
	runCmd.Flags().StringP("image", "i", "", "OCI image to use (overrides config)")
	runCmd.Flags().StringP("workdir", "w", "", "Working directory to mount (overrides config)")
	runCmd.Flags().StringArrayP("mount", "m", nil, "Additional mount source:target[:ro|rw] (repeatable)")
	runCmd.Flags().StringArrayP("env-secret", "e", nil, "Env secret ENV_NAME[:locations] (repeatable)")
	runCmd.Flags().Int("proxy-port", 0, "Port for MITM proxy (0=auto)")
	runCmd.Flags().Bool("no-auto-mounts", false, "Disable automatic config directory mounts")

	rootCmd.AddCommand(runCmd)
}

func runRun(cmd *cobra.Command, args []string) error {
	configPath, _ := cmd.Flags().GetString("config")
	image, _ := cmd.Flags().GetString("image")
	workdir, _ := cmd.Flags().GetString("workdir")
	mountFlags, _ := cmd.Flags().GetStringArray("mount")
	envSecretFlags, _ := cmd.Flags().GetStringArray("env-secret")
	proxyPort, _ := cmd.Flags().GetInt("proxy-port")

	// Load config file.
	cfg, err := config.Load(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Merge CLI flag overrides.
	if err := config.MergeFlags(cfg, image, workdir, mountFlags, envSecretFlags, proxyPort); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Resolve default image if none configured.
	config.ResolveDefaultImage(cfg)

	// Resolve env-based secrets (reads from host env, generates placeholders).
	if err := config.ResolveEnvSecrets(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Apply auto-mounts unless disabled.
	noAutoMounts, _ := cmd.Flags().GetBool("no-auto-mounts")
	if !noAutoMounts {
		cfg.NoAutoMounts = false
	} else {
		cfg.NoAutoMounts = true
	}

	// Validate.
	if err := config.Validate(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Create and start sandbox.
	mgr, err := sandbox.NewManager(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(2)
	}

	if err := mgr.Start(context.Background()); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		errMsg := err.Error()
		if strings.Contains(errMsg, "proxy") || strings.Contains(errMsg, "TLS") || strings.Contains(errMsg, "CA") {
			os.Exit(3)
		}
		os.Exit(2)
	}

	return nil
}
