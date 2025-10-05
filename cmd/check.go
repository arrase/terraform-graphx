package cmd

import (
	"context"
	"fmt"
	"log"
	"terraform-graphx/internal/config"
	"terraform-graphx/internal/neo4j"

	"github.com/spf13/cobra"
)

var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "Validate terraform-graphx configuration and connections",
	Long:  `Validate terraform-graphx configuration and verify connections.`,
}

var checkDatabaseCmd = &cobra.Command{
	Use:   "database",
	Short: "Check Neo4j database connectivity",
	Long: `Verify that terraform-graphx can connect to the Neo4j database using
the credentials from the configuration file (.terraform-graphx.yaml).

This command will:
  1. Load the configuration from .terraform-graphx.yaml
  2. Attempt to connect to the Neo4j database
  3. Verify connectivity
  4. Report the connection status

Example:
  terraform graphx check database`,
	RunE: runCheckDatabase,
}

func runCheckDatabase(cmd *cobra.Command, args []string) error {
	// Load configuration
	log.Println("Loading configuration from .terraform-graphx.yaml...")
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Check if config file exists
	if !config.Exists() {
		fmt.Println("⚠ Warning: No configuration file found.")
		fmt.Println("  Run 'terraform graphx init config' to create one.")
		fmt.Println("  Using default values...")
		fmt.Println()
	}

	// Display connection info (without password)
	fmt.Println("Neo4j Connection Settings:")
	fmt.Printf("  URI:  %s\n", cfg.Neo4j.URI)
	fmt.Printf("  User: %s\n", cfg.Neo4j.User)
	fmt.Println()

	// Validate configuration
	if cfg.Neo4j.Password == "" {
		return fmt.Errorf("neo4j password is not set in configuration file")
	}

	// Create Neo4j client
	log.Printf("Connecting to Neo4j at %s...", cfg.Neo4j.URI)
	ctx := context.Background()

	client, err := neo4j.NewClient(cfg.Neo4j.URI, cfg.Neo4j.User, cfg.Neo4j.Password)
	if err != nil {
		return fmt.Errorf("failed to create neo4j client: %w", err)
	}
	defer client.Close(ctx)

	// Verify connectivity
	log.Println("Verifying connectivity...")
	if err := client.VerifyConnectivity(ctx); err != nil {
		return fmt.Errorf("failed to connect to neo4j: %w", err)
	}

	fmt.Println()
	fmt.Println("✓ Successfully connected to Neo4j database!")
	fmt.Println("  The database is ready to use.")

	return nil
}

func init() {
	rootCmd.AddCommand(checkCmd)
	checkCmd.AddCommand(checkDatabaseCmd)
}
