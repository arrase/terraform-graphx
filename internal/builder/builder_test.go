package builder

import (
	"os"
	"path/filepath"
	"testing"
	"terraform-graphx/internal/parser"
)

func TestBuild(t *testing.T) {
	// Load and parse the sample plan file
	path := filepath.Join("../parser/testdata", "sample_plan.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read test data file: %v", err)
	}
	plan, err := parser.ParseFromData(data)
	if err != nil {
		t.Fatalf("Failed to parse test plan: %v", err)
	}

	// Build the graph
	graph := Build(plan)

	// Assertions for the graph
	if graph == nil {
		t.Fatal("Build returned a nil graph")
	}

	// Check nodes
	if len(graph.Nodes) != 2 {
		t.Errorf("Expected 2 nodes, got %d", len(graph.Nodes))
	}

	nodeAddresses := make(map[string]bool)
	for _, n := range graph.Nodes {
		nodeAddresses[n.ID] = true
	}
	expectedNodes := []string{"null_resource.app", "null_resource.cluster"}
	for _, expected := range expectedNodes {
		if !nodeAddresses[expected] {
			t.Errorf("Expected node with address '%s' was not found", expected)
		}
	}

	// Check edges
	if len(graph.Edges) != 1 {
		t.Fatalf("Expected 1 edge, got %d", len(graph.Edges))
	}

	edge := graph.Edges[0]
	expectedFrom := "null_resource.app"
	expectedTo := "null_resource.cluster"

	if edge.From != expectedFrom {
		t.Errorf("Expected edge 'from' to be '%s', got '%s'", expectedFrom, edge.From)
	}
	if edge.To != expectedTo {
		t.Errorf("Expected edge 'to' to be '%s', got '%s'", expectedTo, edge.To)
	}
	if edge.Relation != "DEPENDS_ON" {
		t.Errorf("Expected edge relation to be 'DEPENDS_ON', got '%s'", edge.Relation)
	}
}