package formatter

import (
	"strings"
	"terraform-graphx/internal/graph"
	"testing"
)

var testGraph = &graph.Graph{
	Nodes: []graph.Node{
		{ID: "aws_vpc.main", Type: "aws_vpc", Provider: "aws", Name: "main"},
		{ID: "aws_subnet.public", Type: "aws_subnet", Provider: "aws", Name: "public"},
	},
	Edges: []graph.Edge{
		{From: "aws_subnet.public", To: "aws_vpc.main", Relation: "DEPENDS_ON"},
	},
}

func TestToCypherTransaction(t *testing.T) {
	query, params := ToCypherTransaction(testGraph)

	// Check the query string
	if !strings.Contains(query, "UNWIND $nodes AS node_data") {
		t.Error("Transactional cypher query missing 'UNWIND $nodes'")
	}
	if !strings.Contains(query, "UNWIND $edges AS edge_data") {
		t.Error("Transactional cypher query missing 'UNWIND $edges'")
	}

	// Check the parameters
	if _, ok := params["nodes"]; !ok {
		t.Error("Parameters map missing 'nodes' key")
	}
	if _, ok := params["edges"]; !ok {
		t.Error("Parameters map missing 'edges' key")
	}

	nodes, _ := params["nodes"].([]map[string]interface{})
	if len(nodes) != 2 {
		t.Errorf("Expected 2 nodes in params, got %d", len(nodes))
	}

	edges, _ := params["edges"].([]map[string]string)
	if len(edges) != 1 {
		t.Errorf("Expected 1 edge in params, got %d", len(edges))
	}
}
