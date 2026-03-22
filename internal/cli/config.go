package cli

import (
	"fmt"
	"os"

	"github.com/codingbox/codingbox/internal/config"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage sandbox configuration",
}

var configInitOutput string

var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Generate a starter configuration file",
	RunE:  runConfigInit,
}

var configValidateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate a sandbox configuration file",
	RunE:  runConfigValidate,
}

func init() {
	configInitCmd.Flags().StringVar(&configInitOutput, "output", "./codingbox.yml", "output path for configuration file")
	configCmd.AddCommand(configInitCmd)
	configCmd.AddCommand(configValidateCmd)
	rootCmd.AddCommand(configCmd)
}

const starterTemplate = `# codingbox.yml - Sandbox configuration
name: my-sandbox
agent: claude

# Host directory to mount as /workspace inside the sandbox
workspace: /path/to/your/project

# Additional directory mounts (optional)
# mounts:
#   - host: /path/on/host
#     sandbox: /path/in/sandbox
#     mode: ro  # ro or rw

# Secret injection (optional)
# Secrets are injected into outbound HTTP requests by the proxy.
# The agent only sees placeholder UUIDs, never real secret values.
# secrets:
#   - name: openai-api-key
#     host: api.openai.com
#     header: Authorization
#     template: "Bearer {secret}"
#     value: sk-proj-your-key-here

# Base Docker image for the agent container (optional)
# base_image: ubuntu:22.04

# Pre-installed tools (optional)
# tools:
#   - node:20
#   - python:3.12

# Proxy configuration (optional)
# proxy:
#   port: 0  # 0 = auto-assign
`

func runConfigInit(cmd *cobra.Command, args []string) error {
	if _, err := os.Stat(configInitOutput); err == nil {
		return fmt.Errorf("codingbox: error: file already exists: %s", configInitOutput)
	}

	if err := os.WriteFile(configInitOutput, []byte(starterTemplate), 0644); err != nil {
		return fmt.Errorf("codingbox: error: writing config: %w", err)
	}

	fmt.Printf("Created %s\n", configInitOutput)
	return nil
}

func runConfigValidate(cmd *cobra.Command, args []string) error {
	_, err := config.LoadSandboxConfig(cfgFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "codingbox: error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Configuration is valid")
	return nil
}
