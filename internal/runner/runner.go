package runner

import (
	"context"
	"encoding/json"
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

// generateGraphData runs `terraform graph` and uses gographviz to convert DOT to JSON.
func generateGraphData(planFile string) ([]byte, error) {
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

	// Parse DOT using gographviz and build graph
	graphAst, err := gographviz.ParseString(string(dotOutput))
	if err != nil {
		return nil, fmt.Errorf("failed to parse DOT output: %w", err)
	}

	// Convert AST to Graph structure
	graph := gographviz.NewGraph()
	if err := gographviz.Analyse(graphAst, graph); err != nil {
		return nil, fmt.Errorf("failed to analyse graph: %w", err)
	}

	// Convert the parsed graph to JSON format
	// The format is compatible with what the parser expects (dot -Tjson format)
	jsonOutput, err := convertGraphToJSON(graph)
	if err != nil {
		return nil, fmt.Errorf("failed to convert graph to JSON: %w", err)
	}

	return jsonOutput, nil
}

// convertGraphToJSON converts a gographviz.Graph to the JSON format expected by the parser
// This mimics the output format of `dot -Tjson`
func convertGraphToJSON(g *gographviz.Graph) ([]byte, error) {
	type jsonNode struct {
		ID    int    `json:"_gvid"`
		Name  string `json:"name"`
		Label string `json:"label,omitempty"`
	}

	type jsonEdge struct {
		Tail int `json:"tail"`
		Head int `json:"head"`
	}

	type jsonGraph struct {
		Objects []jsonNode `json:"objects"`
		Edges   []jsonEdge `json:"edges"`
	}

	// Build node map
	nodeMap := make(map[string]int)
	nodes := []jsonNode{}
	nodeID := 0

	for nodeName, node := range g.Nodes.Lookup {
		// Remove quotes from node name
		cleanName := nodeName
		if len(cleanName) >= 2 && cleanName[0] == '"' && cleanName[len(cleanName)-1] == '"' {
			cleanName = cleanName[1 : len(cleanName)-1]
		}

		label := cleanName
		if node.Attrs != nil {
			if labelAttr, ok := node.Attrs["label"]; ok {
				// Remove quotes from label
				label = labelAttr
				if len(label) >= 2 && label[0] == '"' && label[len(label)-1] == '"' {
					label = label[1 : len(label)-1]
				}
			}
		}

		nodes = append(nodes, jsonNode{
			ID:    nodeID,
			Name:  cleanName,
			Label: label,
		})
		nodeMap[nodeName] = nodeID
		nodeID++
	}

	// Build edges
	edges := []jsonEdge{}
	for _, edge := range g.Edges.Edges {
		if tailID, ok := nodeMap[edge.Src]; ok {
			if headID, ok := nodeMap[edge.Dst]; ok {
				edges = append(edges, jsonEdge{
					Tail: tailID,
					Head: headID,
				})
			}
		}
	}

	result := jsonGraph{
		Objects: nodes,
		Edges:   edges,
	}

	return json.Marshal(result)
}

// handleOutput updates the Neo4j database with the graph data.
func handleOutput(g *graph.Graph, cfg *config.Config) error {
	if !cfg.Update {
		return fmt.Errorf("no operation specified. Use the 'update' command to push data to Neo4j")
	}
	return updateNeo4jDatabase(g, &cfg.Neo4j)
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

func validateNeo4jConfig(cfg *config.Neo4jConfig) error {
	if cfg.URI == "" || cfg.User == "" || cfg.Password == "" {
		return fmt.Errorf("neo4j-uri, neo4j-user, and neo4j-pass are required when using the update command. Please configure them in .terraform-graphx.yaml or pass them as flags")
	}
	return nil
}
