package cmd

import (
	"context"
	"fmt"
	"log"
	"os"
	"terraform-graphx/internal/builder"
	"terraform-graphx/internal/formatter"
	"terraform-graphx/internal/graph"
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

var graphxCmd = &cobra.Command{
	Use:   "graphx",
	Short: "Generate a dependency graph of Terraform resources",
	Long: `terraform-graphx parses Terraform plan output and generates
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
	RunE: runGraphx,
}

func runGraphx(cmd *cobra.Command, args []string) error {
	// Use provided plan file if specified
	if len(args) > 0 {
		planFile = args[0]
	}

	// Parse Terraform plan
	log.Println("Parsing Terraform plan...")
	plan, err := parser.Parse(planFile)
	if err != nil {
		return fmt.Errorf("failed to parse terraform plan: %w", err)
	}

	// Build dependency graph
	log.Println("Building dependency graph...")
	g := builder.Build(plan)

	// Handle Neo4j update or format output
	if update {
		return updateNeo4jDatabase(g)
	}

	return formatAndPrintGraph(g)
}

func updateNeo4jDatabase(g *graph.Graph) error {
	if err := validateNeo4jFlags(); err != nil {
		return err
	}

	log.Printf("Connecting to Neo4j at %s...", neo4jURI)
	ctx := context.Background()

	client, err := neo4j.NewClient(neo4jURI, neo4jUser, neo4jPass)
	if err != nil {
		return fmt.Errorf("failed to create neo4j client: %w", err)
	}
	defer client.Close(ctx)

	if err := client.VerifyConnectivity(ctx); err != nil {
		return fmt.Errorf("failed to connect to neo4j: %w", err)
	}

	log.Println("Updating Neo4j database...")
	if err := client.UpdateGraph(ctx, g); err != nil {
		return fmt.Errorf("failed to update neo4j graph: %w", err)
	}

	log.Println("Successfully updated Neo4j database.")
	return nil
}

func formatAndPrintGraph(g *graph.Graph) error {
	log.Printf("Formatting graph as %s...", format)

	var output string
	var err error

	switch format {
	case "json":
		output, err = formatter.ToJSON(g)
	case "cypher":
		output, err = formatter.ToCypher(g)
	default:
		return fmt.Errorf("invalid output format: %s (valid formats: json, cypher)", format)
	}

	if err != nil {
		return fmt.Errorf("failed to format graph: %w", err)
	}

	fmt.Fprintln(os.Stdout, output)
	return nil
}

func validateNeo4jFlags() error {
	if neo4jURI == "" || neo4jUser == "" || neo4jPass == "" {
		return fmt.Errorf("--neo4j-uri, --neo4j-user, and --neo4j-pass are required when using --update")
	}
	return nil
}

func init() {
	rootCmd.AddCommand(graphxCmd)

	// Output format flags
	graphxCmd.Flags().StringVar(&format, "format", "json", "Output format for the graph (json, cypher)")
	graphxCmd.Flags().StringVar(&planFile, "plan", "", "Path to a terraform plan file (optional)")

	// Neo4j integration flags
	graphxCmd.Flags().BoolVar(&update, "update", false, "Update a Neo4j database with the graph")
	graphxCmd.Flags().StringVar(&neo4jURI, "neo4j-uri", "bolt://localhost:7687", "URI for the Neo4j database")
	graphxCmd.Flags().StringVar(&neo4jUser, "neo4j-user", "neo4j", "Username for the Neo4j database")
	graphxCmd.Flags().StringVar(&neo4jPass, "neo4j-pass", "", "Password for the Neo4j database")
}
