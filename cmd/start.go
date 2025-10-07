package cmd

import (
	"context"
	"fmt"
	"terraform-graphx/internal/config"
	"terraform-graphx/internal/docker"

	"github.com/spf13/cobra"
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start Neo4j database in Docker",
	Long: `Start a Neo4j database container using Docker with the configuration
from .terraform-graphx.yaml file. The container will use the neo4j-data directory
as a volume for data persistence.

This command will:
  - Pull the Neo4j image if not already downloaded
  - Start a Neo4j container in the background
  - Use the credentials from the configuration file
  - Mount the neo4j-data directory as a volume

Example:
  terraform-graphx start`,
	RunE: runStart,
}

func runStart(cmd *cobra.Command, args []string) error {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Start the Neo4j container
	ctx := context.Background()
	return docker.StartContainer(ctx, docker.StartContainerOptions{
		Config: cfg,
	})
}

func init() {
	rootCmd.AddCommand(startCmd)
}
