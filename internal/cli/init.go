package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Generate a default configuration file",
	Long:  "Generate a default codingbox.yaml configuration file in the current directory.",
	RunE:  runInit,
}

func init() {
	initCmd.Flags().String("image", "", "Pre-fill the image field")
	initCmd.Flags().StringArray("env-secret", nil, "Pre-fill env secret entries (repeatable)")
	initCmd.Flags().Bool("force", false, "Overwrite existing config file")

	rootCmd.AddCommand(initCmd)
}

func runInit(cmd *cobra.Command, args []string) error {
	image, _ := cmd.Flags().GetString("image")
	envSecrets, _ := cmd.Flags().GetStringArray("env-secret")
	force, _ := cmd.Flags().GetBool("force")

	cfgPath := filepath.Join(".", "codingbox.yaml")

	if _, err := os.Stat(cfgPath); err == nil && !force {
		return fmt.Errorf("codingbox.yaml already exists. Use --force to overwrite")
	}

	imageVal := `"ubuntu:22.04"`
	if image != "" {
		imageVal = fmt.Sprintf("%q", image)
	}

	content := fmt.Sprintf(`# codingbox configuration
# See: https://github.com/mischa/codingbox

# OCI image to use as the sandbox environment (required)
image: %s

# Host directory to mount as /workspace in the container (default: current directory)
# workdir: "."

# Port for the MITM proxy (0 = auto-assign)
# proxy_port: 0

# Additional directory mounts
# mounts:
#   - source: "/path/to/shared/libs"
#     target: "/libs"
#     mode: "ro"    # ro (read-only) or rw (read-write)
#   - source: "/path/to/output"
#     target: "/output"
#     mode: "rw"

# Secrets: reads value from host environment automatically
# Inside the sandbox, env vars are set to auto-generated placeholders.
# The proxy replaces placeholders with real values in outbound requests.
# secrets:
#   - env: "ANTHROPIC_API_KEY"
#     replace_in: ["headers"]
#   - env: "GITHUB_TOKEN"
#     replace_in: ["headers", "body"]
`, imageVal)

	// Append env secret entries if provided.
	if len(envSecrets) > 0 {
		content += "\nsecrets:\n"
		for _, es := range envSecrets {
			content += fmt.Sprintf("  - env: %q\n    replace_in: [\"headers\", \"body\", \"query\"]\n", es)
		}
	}

	if err := os.WriteFile(cfgPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("writing config file: %w", err)
	}

	fmt.Printf("Created %s\n", cfgPath)
	return nil
}
