package cmd

import (
	"fmt"
	"os"
	"terraform-graphx/internal/config"
	"terraform-graphx/internal/git"

	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize terraform-graphx configuration",
	Long: `Initialize terraform-graphx configuration and settings.

Creates a .terraform-graphx.yaml configuration file in the current directory
with default values and a randomly generated password. Also creates the neo4j-data
directory for Docker volume mounting.

The configuration file will be created with the following default values:
  - neo4j.uri: bolt://localhost:7687
  - neo4j.user: neo4j
  - neo4j.password: (randomly generated)
  - neo4j.docker_image: neo4j:community

Example:
  terraform-graphx init`,
	RunE: runInit,
}

func runInit(cmd *cobra.Command, args []string) error {
	configPath := ".terraform-graphx.yaml"

	// Initialize configuration and data directory
	result, err := config.Initialize(configPath)
	if err != nil {
		return err
	}

	// Print success messages
	fmt.Printf("✓ Created configuration file: %s\n\n", result.ConfigPath)
	fmt.Println("Default configuration:")
	fmt.Printf("  neo4j.uri: %s\n", result.Config.Neo4j.URI)
	fmt.Printf("  neo4j.user: %s\n", result.Config.Neo4j.User)
	fmt.Printf("  neo4j.password: %s\n", result.Config.Neo4j.Password)
	fmt.Printf("  neo4j.docker_image: %s\n\n", result.Config.Neo4j.DockerImage)
	fmt.Printf("✓ Created data directory: %s\n\n", result.DataDir)

	// Attempt to update .gitignore
	entriesToIgnore := []string{".terraform-graphx.yaml", "neo4j-data/"}
	if err := git.UpdateGitignore(entriesToIgnore); err != nil {
		// If gitignore update fails, print a warning but don't fail the command
		fmt.Fprintf(os.Stderr, "Warning: failed to update .gitignore: %v\n", err)
		fmt.Println("Please manually add '.terraform-graphx.yaml' and 'neo4j-data/' to your .gitignore file.")
	}

	return nil
}

func init() {
	rootCmd.AddCommand(initCmd)
}
