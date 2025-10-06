package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"terraform-graphx/internal/config"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/spf13/cobra"
)

const containerName = "terraform-graphx-neo4j"

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

	// Validate config
	if cfg.Neo4j.Password == "" {
		return fmt.Errorf("neo4j password not set in configuration file")
	}

	// Get absolute path to neo4j-data directory
	dataDir, err := filepath.Abs("neo4j-data")
	if err != nil {
		return fmt.Errorf("failed to get absolute path for neo4j-data: %w", err)
	}

	// Check neo4j-data directory
	hasExistingData := false

	if _, err := os.Stat(dataDir); err == nil {
		// Check if there's existing Neo4j data (dbms directory indicates initialized database)
		dbmsDir := filepath.Join(dataDir, "dbms")
		if _, err := os.Stat(dbmsDir); err == nil {
			hasExistingData = true
		}
	} else if os.IsNotExist(err) {
		return fmt.Errorf("neo4j-data directory does not exist, please run 'terraform-graphx init' first")
	}

	// Warn if using existing data
	if hasExistingData {
		fmt.Println("⚠ Warning: Existing Neo4j data detected in neo4j-data directory")
		fmt.Println("  Neo4j will use the password from the existing database, NOT from the config file.")
		fmt.Println("  If you don't know the existing password, you can:")
		fmt.Printf("    1. Delete the neo4j-data directory and run 'terraform-graphx start' again\n")
		fmt.Printf("    2. Or update the password in .terraform-graphx.yaml to match the existing database\n")
		fmt.Println()
	}

	// Create Docker client
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return fmt.Errorf("failed to create Docker client: %w", err)
	}
	defer cli.Close()

	// Check if container already exists
	containers, err := cli.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		return fmt.Errorf("failed to list containers: %w", err)
	}

	for _, c := range containers {
		for _, name := range c.Names {
			if name == "/"+containerName {
				if c.State == "running" {
					return fmt.Errorf("container %s is already running", containerName)
				}
				// Remove stopped container
				fmt.Printf("Removing stopped container %s...\n", containerName)
				if err := cli.ContainerRemove(ctx, c.ID, container.RemoveOptions{Force: true}); err != nil {
					return fmt.Errorf("failed to remove stopped container: %w", err)
				}
			}
		}
	}

	// Pull image if not present
	fmt.Printf("Checking for Docker image %s...\n", cfg.Neo4j.DockerImage)
	_, _, err = cli.ImageInspectWithRaw(ctx, cfg.Neo4j.DockerImage)
	if err != nil {
		fmt.Printf("Pulling image %s...\n", cfg.Neo4j.DockerImage)
		reader, err := cli.ImagePull(ctx, cfg.Neo4j.DockerImage, image.PullOptions{})
		if err != nil {
			return fmt.Errorf("failed to pull image: %w", err)
		}
		defer reader.Close()
		io.Copy(os.Stdout, reader)
	} else {
		fmt.Printf("✓ Image %s already present\n", cfg.Neo4j.DockerImage)
	}

	// Create container
	fmt.Printf("Creating Neo4j container...\n")

	containerConfig := &container.Config{
		Image: cfg.Neo4j.DockerImage,
		Env: []string{
			fmt.Sprintf("NEO4J_AUTH=%s/%s", cfg.Neo4j.User, cfg.Neo4j.Password),
		},
		ExposedPorts: nat.PortSet{
			"7474/tcp": struct{}{},
			"7687/tcp": struct{}{},
		},
	}

	hostConfig := &container.HostConfig{
		PortBindings: nat.PortMap{
			"7474/tcp": []nat.PortBinding{{HostIP: "0.0.0.0", HostPort: "7474"}},
			"7687/tcp": []nat.PortBinding{{HostIP: "0.0.0.0", HostPort: "7687"}},
		},
		Binds: []string{
			fmt.Sprintf("%s:/data", dataDir),
		},
	}

	resp, err := cli.ContainerCreate(ctx, containerConfig, hostConfig, nil, nil, containerName)
	if err != nil {
		return fmt.Errorf("failed to create container: %w", err)
	}

	// Start container
	if err := cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return fmt.Errorf("failed to start container: %w", err)
	}

	fmt.Printf("✓ Neo4j container started successfully\n")
	fmt.Printf("  Container ID: %s\n", resp.ID[:12])
	fmt.Printf("  Container Name: %s\n", containerName)
	fmt.Printf("  Data Directory: %s\n", dataDir)
	fmt.Printf("  Neo4j Browser: http://localhost:7474\n")
	fmt.Printf("  Bolt URI: %s\n", cfg.Neo4j.URI)
	fmt.Printf("\nWaiting for Neo4j to be ready (this may take a few seconds)...\n")

	// Give Neo4j some time to start
	time.Sleep(5 * time.Second)

	fmt.Printf("✓ Neo4j should now be ready\n")
	fmt.Printf("\nYou can verify the connection with:\n")
	fmt.Printf("  terraform-graphx check database\n")

	return nil
}

func init() {
	rootCmd.AddCommand(startCmd)
}
