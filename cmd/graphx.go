/*
Copyright © 2025 Jules
*/
package cmd

import (
	"context"
	"fmt"
	"log"
	"os"
	"terraform-graphx/internal/builder"
	"terraform-graphx/internal/formatter"
	"terraform-graphx/internal/neo4j"
	"terraform-graphx/internal/parser"

	"github.com/spf13/cobra"
)

var (
	format    string
	planFile  string
	update    bool
	neo4jURI  string
	neo4jUser string
	neo4jPass string
)

// graphxCmd represents the graphx command
var graphxCmd = &cobra.Command{
	Use:   "graphx",
	Short: "Generate a dependency graph of Terraform resources.",
	Long: `terraform-graphx is a CLI tool that integrates with Terraform to generate
a dependency graph of your infrastructure.

It works by parsing the output of 'terraform show -json' and can output
the graph in various formats or update a Neo4j database.

Examples:
  # Output the graph as JSON
  terraform graphx --format=json > graph.json

  # Output the graph as Cypher statements
  terraform graphx --format=cypher > graph.cypher

  # Update a Neo4j database with the current infrastructure state
  terraform graphx --update --neo4j-uri=bolt://localhost:7687 --neo4j-user=neo4j --neo4j-pass=secret`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// If a plan file is specified, use it. Otherwise, generate one.
		if len(args) > 0 {
			planFile = args[0]
		}

		log.Println("Parsing Terraform plan...")
		plan, err := parser.Parse(planFile)
		if err != nil {
			return fmt.Errorf("error parsing terraform plan: %w", err)
		}

		log.Println("Building dependency graph...")
		graph := builder.Build(plan)

		if update {
			if neo4jURI == "" || neo4jUser == "" || neo4jPass == "" {
				return fmt.Errorf("--neo4j-uri, --neo4j-user, and --neo4j-pass must be provided when using --update")
			}

			log.Printf("Connecting to Neo4j at %s...", neo4jURI)
			ctx := context.Background()
			client, err := neo4j.NewClient(neo4jURI, neo4jUser, neo4jPass)
			if err != nil {
				return fmt.Errorf("error creating neo4j client: %w", err)
			}
			defer client.Close(ctx)

			if err := client.VerifyConnectivity(ctx); err != nil {
				return fmt.Errorf("error verifying neo4j connection: %w", err)
			}

			log.Println("Updating Neo4j database...")
			if err := client.UpdateGraph(ctx, graph); err != nil {
				return fmt.Errorf("error updating neo4j graph: %w", err)
			}

			log.Println("Successfully updated Neo4j database.")

		} else {
			var output string
			var err error

			log.Printf("Formatting graph as %s...", format)
			switch format {
			case "json":
				output, err = formatter.ToJSON(graph)
			case "cypher":
				output, err = formatter.ToCypher(graph)
			default:
				return fmt.Errorf("invalid output format: %s. valid formats are 'json' and 'cypher'", format)
			}

			if err != nil {
				return fmt.Errorf("error formatting graph: %w", err)
			}

			fmt.Fprintln(os.Stdout, output)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(graphxCmd)

	// Flags for output formatting
	graphxCmd.Flags().StringVar(&format, "format", "json", "Output format for the graph (json, cypher).")

	// Optional plan file input
	graphxCmd.Flags().StringVar(&planFile, "plan", "", "Path to a terraform plan file (e.g., plan.out). If not provided, a plan will be generated.")

	// Flags for Neo4j integration
	graphxCmd.Flags().BoolVar(&update, "update", false, "Update a Neo4j database with the graph.")
	graphxCmd.Flags().StringVar(&neo4jURI, "neo4j-uri", "bolt://localhost:7687", "URI for the Neo4j database.")
	graphxCmd.Flags().StringVar(&neo4jUser, "neo4j-user", "neo4j", "Username for the Neo4j database.")
	graphxCmd.Flags().StringVar(&neo4jPass, "neo4j-pass", "", "Password for the Neo4j database.")
}