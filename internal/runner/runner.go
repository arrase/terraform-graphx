package runner

import (
	"context"
	"fmt"
	"log"
	"os"
	"terraform-graphx/internal/builder"
	"terraform-graphx/internal/config"
	"terraform-graphx/internal/formatter"
	"terraform-graphx/internal/graph"
	"terraform-graphx/internal/neo4j"
	"terraform-graphx/internal/parser"
)

// Run executes the main logic of terraform-graphx.
func Run(cfg *config.Config) error {
	// Parse Terraform plan
	log.Println("Parsing Terraform plan...")
	plan, err := parser.Parse(cfg.PlanFile)
	if err != nil {
		return fmt.Errorf("failed to parse terraform plan: %w", err)
	}

	// Build dependency graph
	log.Println("Building dependency graph...")
	g := builder.Build(plan)

	// Handle output
	return handleOutput(g, cfg)
}

// handleOutput decides whether to update Neo4j or format and print the graph.
func handleOutput(g *graph.Graph, cfg *config.Config) error {
	if cfg.Update {
		return updateNeo4jDatabase(g, &cfg.Neo4j)
	}
	return formatAndPrintGraph(g, cfg.Format)
}

func updateNeo4jDatabase(g *graph.Graph, neo4jCfg *config.Neo4jConfig) error {
	if err := validateNeo4jConfig(neo4jCfg); err != nil {
		return err
	}

	log.Printf("Connecting to Neo4j at %s...", neo4jCfg.URI)
	ctx := context.Background()

	client, err := neo4j.NewClient(neo4jCfg.URI, neo4jCfg.User, neo4jCfg.Password)
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

func formatAndPrintGraph(g *graph.Graph, format string) error {
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

func validateNeo4jConfig(cfg *config.Neo4jConfig) error {
	if cfg.URI == "" || cfg.User == "" || cfg.Password == "" {
		return fmt.Errorf("neo4j-uri, neo4j-user, and neo4j-pass are required when using --update")
	}
	return nil
}