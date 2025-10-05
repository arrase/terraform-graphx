package formatter

import (
	"strings"
	"testing"
	"terraform-graphx/internal/graph"
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

func TestToJSON(t *testing.T) {
	jsonString, err := ToJSON(testGraph)
	if err != nil {
		t.Fatalf("ToJSON failed: %v", err)
	}

	// Basic check to ensure the output looks like JSON and contains our node IDs
	if !strings.Contains(jsonString, `"id": "aws_vpc.main"`) {
		t.Errorf("JSON output does not contain expected node 'aws_vpc.main'")
	}
	if !strings.Contains(jsonString, `"from": "aws_subnet.public"`) {
		t.Errorf("JSON output does not contain expected edge from 'aws_subnet.public'")
	}
}

func TestToCypher(t *testing.T) {
	cypherString, err := ToCypher(testGraph)
	if err != nil {
		t.Fatalf("ToCypher failed: %v", err)
	}

	// Check for node creation
	expectedNode1 := "MERGE (n:Resource {id: 'aws_vpc.main'})"
	if !strings.Contains(cypherString, expectedNode1) {
		t.Errorf("Cypher output missing expected node statement: %s", expectedNode1)
	}

	// Check for edge creation
	expectedEdge := "MATCH (from:Resource {id: 'aws_subnet.public'}), (to:Resource {id: 'aws_vpc.main'})\nMERGE (from)-[:DEPENDS_ON]->(to);"
	if !strings.Contains(cypherString, expectedEdge) {
		t.Errorf("Cypher output missing expected edge statement: %s", expectedEdge)
	}
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