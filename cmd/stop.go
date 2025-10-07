package cmd

import (
	"context"
	"terraform-graphx/internal/docker"

	"github.com/spf13/cobra"
)

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop and remove the Neo4j Docker container",
	Long: `Stop and remove the Neo4j Docker container started with 'terraform-graphx start'.

This command will:
  - Stop the running Neo4j container
  - Remove the container
  - Preserve the data in the neo4j-data directory

Example:
  terraform-graphx stop`,
	RunE: runStop,
}

func runStop(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	return docker.StopContainer(ctx)
}

func init() {
	rootCmd.AddCommand(stopCmd)
}
