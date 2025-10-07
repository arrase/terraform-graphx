package runner

import (
	"context"
	"fmt"
	"log"
	"os/exec"
	"terraform-graphx/internal/config"
	"terraform-graphx/internal/graph"
	"terraform-graphx/internal/neo4j"
	graphparser "terraform-graphx/internal/parser"

	"github.com/awalterschulze/gographviz"
)

// Run executes the main logic of terraform-graphx.
func Run(cfg *config.Config) error {
	// Validate Neo4j configuration early
	if err := validateNeo4jConfig(&cfg.Neo4j); err != nil {
		return err
	}

	// Generate and parse Terraform graph
	log.Println("Generating Terraform graph...")
	dotGraph, err := generateTerraformGraph(cfg.PlanFile)
	if err != nil {
		return fmt.Errorf("failed to generate graph data: %w", err)
	}

	// Parse the graph data directly from gographviz
	log.Println("Parsing graph data...")
	g, err := graphparser.ParseGraph(dotGraph)
	if err != nil {
		return fmt.Errorf("failed to parse graph data: %w", err)
	}

	// Update Neo4j database
	return updateNeo4jDatabase(g, &cfg.Neo4j)
}

// generateTerraformGraph runs `terraform graph` and parses the DOT output.
func generateTerraformGraph(planFile string) (*gographviz.Graph, error) {
	var graphArgs []string
	if planFile != "" {
		graphArgs = append(graphArgs, "-plan="+planFile)
	}

	terraformGraphCmd := exec.Command("terraform", append([]string{"graph"}, graphArgs...)...)

	// Get DOT output from terraform graph
	dotOutput, err := terraformGraphCmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("terraform graph command failed: %w - %s", err, string(dotOutput))
	}

	// Parse DOT using gographviz
	graphAst, err := gographviz.ParseString(string(dotOutput))
	if err != nil {
		return nil, fmt.Errorf("failed to parse DOT output: %w", err)
	}

	// Convert AST to Graph structure
	dotGraph := gographviz.NewGraph()
	if err := gographviz.Analyse(graphAst, dotGraph); err != nil {
		return nil, fmt.Errorf("failed to analyse graph: %w", err)
	}

	return dotGraph, nil
}

func updateNeo4jDatabase(g *graph.Graph, neo4jCfg *config.Neo4jConfig) error {
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

func validateNeo4jConfig(cfg *config.Neo4jConfig) error {
	if cfg.URI == "" || cfg.User == "" || cfg.Password == "" {
		return fmt.Errorf("neo4j-uri, neo4j-user, and neo4j-pass are required when using the update command. Please configure them in .terraform-graphx.yaml or pass them as flags")
	}
	return nil
}
