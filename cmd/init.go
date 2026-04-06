package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/begray/agh/internal/config"
	"github.com/begray/agh/internal/project"
)

var (
	forceInit  bool
	globalInit bool
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize .agh/ directory with default config",
	Long:  "Initialize project config (.agh/config.toml) or global config (~/.config/agh/config.toml).",
	Args:  cobra.NoArgs,
	RunE:  runInit,
}

func init() {
	initCmd.Flags().BoolVarP(&forceInit, "force", "f", false, "Overwrite existing config")
	initCmd.Flags().BoolVarP(&globalInit, "global", "g", false, "Initialize global config (~/.config/agh/config.toml)")
	rootCmd.AddCommand(initCmd)
}

func runInit(cmd *cobra.Command, args []string) error {
	if globalInit {
		return runInitGlobal()
	}
	return runInitProject()
}

func runInitGlobal() error {
	dir := config.GlobalConfigDir()
	configPath := config.GlobalConfigPath()

	if _, err := os.Stat(configPath); err == nil && !forceInit {
		fmt.Printf("Global config already exists: %s\n", configPath)
		fmt.Println("Use --force to overwrite")
		return nil
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating global config dir: %w", err)
	}

	if err := config.WriteGlobalDefault(configPath); err != nil {
		return err
	}

	fmt.Printf("Global config written to %s\n", configPath)
	fmt.Println("Edit this file to set your personal defaults (ai_tool, terminal, etc.)")
	return nil
}

func runInitProject() error {
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
