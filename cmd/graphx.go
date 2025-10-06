package cmd

import (
	"terraform-graphx/internal/config"
	"terraform-graphx/internal/runner"

	"github.com/spf13/cobra"
)

var graphxCmd = &cobra.Command{
	Use:   "graphx [plan_file]",
	Short: "Generate a dependency graph of Terraform resources",
	Long: `terraform-graphx generates a dependency graph of your Terraform
resources by invoking 'terraform graph' and converting the DOT output to JSON
using the go-graphviz library. The resulting graph can be emitted as JSON or Cypher, or
optionally pushed to a Neo4j database.

Examples:
	# Read a Terraform plan and output JSON graph
	terraform-graphx plan.tf > graph.json

  # Output the graph as Cypher statements
	terraform-graphx --format=cypher > graph.cypher

  # Update a Neo4j database with the current infrastructure state
	terraform-graphx --update --neo4j-uri=bolt://localhost:7687 --neo4j-user=neo4j --neo4j-pass=secret`,
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
	graphxCmd.Hidden = true
	registerGraphFlags(graphxCmd)
}

func registerGraphFlags(cmd *cobra.Command) {
	// Output format flags
	cmd.Flags().String("format", "json", "Output format for the graph (json, cypher)")
	cmd.Flags().String("plan", "", "Path to a terraform plan file (optional)")

	// Neo4j integration flags
	cmd.Flags().Bool("update", false, "Update a Neo4j database with the graph")
	cmd.Flags().String("neo4j-uri", "bolt://localhost:7687", "URI for the Neo4j database")
	cmd.Flags().String("neo4j-user", "neo4j", "Username for the Neo4j database")
	cmd.Flags().String("neo4j-pass", "", "Password for the Neo4j database")
}
