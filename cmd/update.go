package cmd

import (
	"terraform-graphx/internal/config"
	"terraform-graphx/internal/runner"

	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use:   "update [plan_file]",
	Short: "Update a Neo4j database with the Terraform dependency graph",
	Long: `terraform-graphx update generates a dependency graph of your Terraform
resources by invoking 'terraform graph' and pushes the resulting graph to a Neo4j database.

The graph is stored as nodes (resources) and relationships (dependencies) in Neo4j,
allowing you to query and visualize your infrastructure dependencies.`,
	RunE: runUpdate,
}

func runUpdate(cmd *cobra.Command, args []string) error {
	cfg, err := config.LoadAndMerge(cmd, args)
	if err != nil {
		return err
	}

	return runner.Run(cfg)
}

func init() {
	rootCmd.AddCommand(updateCmd)

	updateCmd.Flags().String("plan", "", "Path to a terraform plan file (optional)")
	updateCmd.Flags().String("neo4j-uri", "bolt://localhost:7687", "URI for the Neo4j database")
	updateCmd.Flags().String("neo4j-user", "neo4j", "Username for the Neo4j database")
	updateCmd.Flags().String("neo4j-pass", "", "Password for the Neo4j database")
}
