package runner

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"terraform-graphx/internal/config"
	"terraform-graphx/internal/formatter"
	"terraform-graphx/internal/graph"
	"terraform-graphx/internal/neo4j"
	graphparser "terraform-graphx/internal/parser"
)

// Run executes the main logic of terraform-graphx.
func Run(cfg *config.Config) error {
	// Generate graph data using `terraform graph`
	log.Println("Generating Terraform graph...")
	graphData, err := generateGraphData(cfg.PlanFile)
	if err != nil {
		return fmt.Errorf("failed to generate graph data: %w", err)
	}

	// Parse the graph data
	log.Println("Parsing graph data...")
	g, err := graphparser.ParseGraph(graphData)
	if err != nil {
		return fmt.Errorf("failed to parse graph data: %w", err)
	}

	// Handle output
	return handleOutput(g, cfg)
}

// generateGraphData runs `terraform graph` and `dot` to get a JSON representation of the graph.
func generateGraphData(planFile string) ([]byte, error) {
	var graphArgs []string
	if planFile != "" {
		graphArgs = append(graphArgs, "-plan="+planFile)
	}

	terraformGraphCmd := exec.Command("terraform", append([]string{"graph"}, graphArgs...)...)
	dotCmd := exec.Command("dot", "-Tjson")

	var dotStdout, dotStderr bytes.Buffer
	dotCmd.Stdout = &dotStdout
	dotCmd.Stderr = &dotStderr

	pipe, err := terraformGraphCmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create pipe: %w", err)
	}
	dotCmd.Stdin = pipe

	if err := terraformGraphCmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start terraform graph command: %w", err)
	}

	if err := dotCmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start dot command: %w", err)
	}

	if err := terraformGraphCmd.Wait(); err != nil {
		return nil, fmt.Errorf("terraform graph command failed: %w", err)
	}

	if err := dotCmd.Wait(); err != nil {
		return nil, fmt.Errorf("dot command failed: %w - %s", err, dotStderr.String())
	}

	return dotStdout.Bytes(), nil
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
