package cli

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/mischa/codingbox/internal/config"
	"github.com/mischa/codingbox/internal/models"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage central per-directory configurations",
	Long:  "Register, list, update, and remove sandbox configurations for directories in the central store (~/.codingbox/directories.yaml).",
}

var configSetCmd = &cobra.Command{
	Use:   "set",
	Short: "Register or update a directory configuration",
	RunE:  runConfigSet,
}

var configListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all registered directory configurations",
	RunE:  runConfigList,
}

var configRemoveCmd = &cobra.Command{
	Use:   "remove",
	Short: "Remove a directory configuration",
	RunE:  runConfigRemove,
}

func init() {
	configSetCmd.Flags().StringP("dir", "d", ".", "Target directory (default: current directory)")
	configSetCmd.Flags().StringP("image", "i", "", "OCI image")
	configSetCmd.Flags().StringArrayP("mount", "m", nil, "Mount source:target[:ro|rw] (repeatable)")
	configSetCmd.Flags().StringArrayP("env-secret", "e", nil, "Env secret ENV_NAME[:locations] (repeatable)")
	configSetCmd.Flags().Int("proxy-port", 0, "Proxy port (0=auto)")

	configRemoveCmd.Flags().StringP("dir", "d", ".", "Target directory (default: current directory)")

	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configListCmd)
	configCmd.AddCommand(configRemoveCmd)
	rootCmd.AddCommand(configCmd)
}

func runConfigSet(cmd *cobra.Command, args []string) error {
	dir, _ := cmd.Flags().GetString("dir")
	image, _ := cmd.Flags().GetString("image")
	mountFlags, _ := cmd.Flags().GetStringArray("mount")
	envSecretFlags, _ := cmd.Flags().GetStringArray("env-secret")
	proxyPort, _ := cmd.Flags().GetInt("proxy-port")

	canonical, err := config.CanonicalDir(dir)
	if err != nil {
		return fmt.Errorf("resolving directory: %w", err)
	}

	store := config.NewDirectoryConfigStore(config.DefaultStorePath())
	if err := store.Load(); err != nil {
		return err
	}

	// Start from existing entry if present, otherwise empty.
	cfg, ok := store.Get(canonical)
	if !ok {
		cfg = &models.SandboxConfig{}
	}

	// Apply flags.
	if image != "" {
		cfg.Image = image
	}
	if proxyPort != 0 {
		cfg.ProxyPort = proxyPort
	}
	for _, mf := range mountFlags {
		m, err := config.ParseMountFlag(mf)
		if err != nil {
			return err
		}
		cfg.Mounts = append(cfg.Mounts, m)
	}
	for _, ef := range envSecretFlags {
		s, err := config.ParseEnvSecretFlag(ef)
		if err != nil {
			return err
		}
		cfg.Secrets = append(cfg.Secrets, s)
	}
	store.Set(canonical, *cfg)
	if err := store.Save(); err != nil {
		return err
	}

	action := "Created"
	if ok {
		action = "Updated"
	}
	fmt.Printf("%s config for %s\n", action, canonical)
	if cfg.Image != "" {
		fmt.Printf("  image: %s\n", cfg.Image)
	}
	if len(cfg.Secrets) > 0 {
		fmt.Printf("  secrets: %d\n", len(cfg.Secrets))
	}
	if len(cfg.Mounts) > 0 {
		fmt.Printf("  mounts: %d\n", len(cfg.Mounts))
	}
	return nil
}

func runConfigList(cmd *cobra.Command, args []string) error {
	store := config.NewDirectoryConfigStore(config.DefaultStorePath())
	if err := store.Load(); err != nil {
		return err
	}

	dirs := store.List()
	if len(dirs) == 0 {
		fmt.Println("No directory configurations registered.")
		fmt.Println("Register one with: codingbox config set --image <image>")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, "DIRECTORY\tIMAGE\tSECRETS\tMOUNTS")
	for dir, cfg := range dirs {
		fmt.Fprintf(w, "%s\t%s\t%d\t%d\n", dir, cfg.Image, len(cfg.Secrets), len(cfg.Mounts))
	}
	w.Flush()
	return nil
}

func runConfigRemove(cmd *cobra.Command, args []string) error {
	dir, _ := cmd.Flags().GetString("dir")

	canonical, err := config.CanonicalDir(dir)
	if err != nil {
		return fmt.Errorf("resolving directory: %w", err)
	}

	store := config.NewDirectoryConfigStore(config.DefaultStorePath())
	if err := store.Load(); err != nil {
		return err
	}

	if !store.Remove(canonical) {
		return fmt.Errorf("no configuration found for %s", canonical)
	}

	if err := store.Save(); err != nil {
		return err
	}

	fmt.Printf("Removed config for %s\n", canonical)
	return nil
}
