package cli

import (
	"fmt"

	"github.com/mischa/codingbox/internal/config"
	"github.com/spf13/cobra"
)

var configSetDefaultCmd = &cobra.Command{
	Use:   "set-default",
	Short: "Set the global default sandbox image",
	RunE:  runConfigSetDefault,
}

var configShowDefaultCmd = &cobra.Command{
	Use:   "show-default",
	Short: "Show the current default sandbox image",
	RunE:  runConfigShowDefault,
}

func init() {
	configSetDefaultCmd.Flags().StringP("image", "i", "", "Default image to use when none is configured")
	configCmd.AddCommand(configSetDefaultCmd)
	configCmd.AddCommand(configShowDefaultCmd)
}

func runConfigSetDefault(cmd *cobra.Command, args []string) error {
	image, _ := cmd.Flags().GetString("image")
	if image == "" {
		return fmt.Errorf("--image is required")
	}

	store := config.NewDirectoryConfigStore(config.DefaultStorePath())
	if err := store.Load(); err != nil {
		return err
	}

	store.Defaults.DefaultImage = image
	if err := store.Save(); err != nil {
		return err
	}

	fmt.Printf("Default image set to %s\n", image)
	return nil
}

func runConfigShowDefault(cmd *cobra.Command, args []string) error {
	store := config.NewDirectoryConfigStore(config.DefaultStorePath())
	if err := store.Load(); err != nil {
		return err
	}

	image := store.Defaults.DefaultImage
	if image == "" {
		image = config.DefaultSandboxImage
	}
	fmt.Println(image)
	return nil
}
