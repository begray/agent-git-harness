package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/vvinogradov/agh/internal/config"
	"github.com/vvinogradov/agh/internal/project"
)

var forceInit bool

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize .agh/ directory with default config",
	Args:  cobra.NoArgs,
	RunE:  runInit,
}

func init() {
	initCmd.Flags().BoolVarP(&forceInit, "force", "f", false, "Overwrite existing config")
	rootCmd.AddCommand(initCmd)
}

func runInit(cmd *cobra.Command, args []string) error {
	proj, err := project.Detect()
	if err != nil {
		return fmt.Errorf("not in a git project: %w", err)
	}

	configPath := filepath.Join(proj.AghDir, "config.toml")
	if _, err := os.Stat(configPath); err == nil && !forceInit {
		fmt.Printf("Config already exists: %s\n", configPath)
		fmt.Println("Use --force to overwrite")
		return nil
	}

	if err := proj.InitAghDir(); err != nil {
		return err
	}

	if err := config.WriteDefault(configPath); err != nil {
		return err
	}

	fmt.Printf("Initialized %s\n", proj.AghDir)
	fmt.Printf("Config written to %s\n", configPath)
	return nil
}
