package cmd

import (
	"terraform-graphx/internal/config"
	"terraform-graphx/internal/runner"

	"github.com/spf13/cobra"
)

var graphxCmd = &cobra.Command{
	Use:   "graphx [plan_file]",
	Short: "Generate a dependency graph of Terraform resources",
	Long: `terraform-graphx parses Terraform plan output and generates
a dependency graph of your infrastructure.

It works by parsing the output of 'terraform show -json' and can output
the graph in various formats or update a Neo4j database.

Examples:
  # Output the graph as JSON from stdin
  terraform show -json . | terraform graphx

  # Output the graph as JSON from a plan file
  terraform graphx --format=json > graph.json

  # Output the graph as Cypher statements
  terraform graphx --format=cypher > graph.cypher

  # Update a Neo4j database with the current infrastructure state
  terraform graphx --update --neo4j-uri=bolt://localhost:7687 --neo4j-user=neo4j --neo4j-pass=secret`,
	RunE: runGraphx,
}

func runGraphx(cmd *cobra.Command, args []string) error {
	cfg, err := config.LoadAndMerge(cmd, args)
	if err != nil {
		return err
	}

	return runner.Run(cfg)
}

func init() {
	rootCmd.AddCommand(graphxCmd)

	// Output format flags
	graphxCmd.Flags().String("format", "json", "Output format for the graph (json, cypher)")
	graphxCmd.Flags().String("plan", "", "Path to a terraform plan file (optional)")

	// Neo4j integration flags
	graphxCmd.Flags().Bool("update", false, "Update a Neo4j database with the graph")
	graphxCmd.Flags().String("neo4j-uri", "bolt://localhost:7687", "URI for the Neo4j database")
	graphxCmd.Flags().String("neo4j-user", "neo4j", "Username for the Neo4j database")
	graphxCmd.Flags().String("neo4j-pass", "", "Password for the Neo4j database")
}