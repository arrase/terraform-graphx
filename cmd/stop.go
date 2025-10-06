package cmd

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
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
	// Create Docker client
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return fmt.Errorf("failed to create Docker client: %w", err)
	}
	defer cli.Close()

	// Check if container exists
	containers, err := cli.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		return fmt.Errorf("failed to list containers: %w", err)
	}

	containerFound := false
	var containerID string

	for _, c := range containers {
		for _, name := range c.Names {
			if name == "/"+containerName {
				containerFound = true
				containerID = c.ID
				break
			}
		}
		if containerFound {
			break
		}
	}

	if !containerFound {
		return fmt.Errorf("container %s not found", containerName)
	}

	// Stop container
	fmt.Printf("Stopping container %s...\n", containerName)
	timeout := 10 // seconds
	if err := cli.ContainerStop(ctx, containerID, container.StopOptions{Timeout: &timeout}); err != nil {
		// Container might already be stopped, try to remove anyway
		fmt.Printf("Warning: failed to stop container: %v\n", err)
	} else {
		fmt.Printf("✓ Container stopped\n")
	}

	// Remove container
	fmt.Printf("Removing container %s...\n", containerName)
	if err := cli.ContainerRemove(ctx, containerID, container.RemoveOptions{Force: true}); err != nil {
		return fmt.Errorf("failed to remove container: %w", err)
	}

	fmt.Printf("✓ Container %s removed successfully\n", containerName)
	fmt.Printf("\nNote: Data has been preserved in the neo4j-data directory\n")

	return nil
}

func init() {
	rootCmd.AddCommand(stopCmd)
}
