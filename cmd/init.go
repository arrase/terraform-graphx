package cmd

import (
	"fmt"
	"os"
	"terraform-graphx/internal/config"

	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize terraform-graphx configuration",
	Long:  `Initialize terraform-graphx configuration and settings.`,
}

var initConfigCmd = &cobra.Command{
	Use:   "config",
	Short: "Create a configuration file with default values",
	Long: `Create a .terraform-graphx.yaml configuration file in the current directory
with default values. You can then edit this file to set your Neo4j credentials.

The configuration file will be created with the following default values:
  - neo4j.uri: bolt://localhost:7687
  - neo4j.user: neo4j
  - neo4j.password: (empty, you must set this)

Example:
  terraform graphx init config`,
	RunE: runInitConfig,
}

func runInitConfig(cmd *cobra.Command, args []string) error {
	configPath := ".terraform-graphx.yaml"

	// Check if config file already exists
	if _, err := os.Stat(configPath); err == nil {
		return fmt.Errorf("configuration file already exists at %s", configPath)
	}

	// Create default config
	cfg := config.DefaultConfig()

	// Save to file
	if err := config.Save(cfg, configPath); err != nil {
		return fmt.Errorf("failed to create config file: %w", err)
	}

	fmt.Printf("✓ Created configuration file: %s\n\n", configPath)
	fmt.Println("Default configuration:")
	fmt.Printf("  neo4j.uri: %s\n", cfg.Neo4j.URI)
	fmt.Printf("  neo4j.user: %s\n", cfg.Neo4j.User)
	fmt.Printf("  neo4j.password: (empty)\n\n")
	fmt.Println("Please edit this file and set your Neo4j password.")
	fmt.Println("\nNote: This file contains sensitive credentials and should not be committed to version control.")
	fmt.Println("Add '.terraform-graphx.yaml' to your .gitignore file.")

	return nil
}

func init() {
	rootCmd.AddCommand(initCmd)
	initCmd.AddCommand(initConfigCmd)
}
