package parser

import (
	"testing"

	"github.com/awalterschulze/gographviz"
)

func TestParseGraph(t *testing.T) {
	// Sample DOT output similar to what Terraform generates
	dotString := `digraph G {
		rankdir = "RL";
		node [shape = rect, fontname = "sans-serif"];
		"null_resource.app" [label="null_resource.app"];
		"null_resource.cluster" [label="null_resource.cluster"];
		"null_resource.app" -> "null_resource.cluster";
	}`

	// Parse DOT string using gographviz
	graphAst, err := gographviz.ParseString(dotString)
	if err != nil {
		t.Fatalf("Failed to parse DOT string: %v", err)
	}

	dotGraph := gographviz.NewGraph()
	if err := gographviz.Analyse(graphAst, dotGraph); err != nil {
		t.Fatalf("Failed to analyse graph: %v", err)
	}

	// Test our parser
	g, err := ParseGraph(dotGraph)
	if err != nil {
		t.Fatalf("ParseGraph failed: %v", err)
	}

	// Verify nodes
	if len(g.Nodes) != 2 {
		t.Errorf("Expected 2 nodes, got %d", len(g.Nodes))
	}

	// Verify edges
	if len(g.Edges) != 1 {
		t.Errorf("Expected 1 edge, got %d", len(g.Edges))
	}

	// Check specific node
	foundApp := false
	for _, node := range g.Nodes {
		if node.ID == "null_resource.app" {
			foundApp = true
			if node.Type != "null_resource" {
				t.Errorf("Expected type 'null_resource', got '%s'", node.Type)
			}
			if node.Name != "app" {
				t.Errorf("Expected name 'app', got '%s'", node.Name)
			}
		}
	}
	if !foundApp {
		t.Error("Did not find null_resource.app node")
	}

	// Check edge
	if len(g.Edges) > 0 {
		edge := g.Edges[0]
		if edge.From != "null_resource.app" {
			t.Errorf("Expected edge from 'null_resource.app', got '%s'", edge.From)
		}
		if edge.To != "null_resource.cluster" {
			t.Errorf("Expected edge to 'null_resource.cluster', got '%s'", edge.To)
		}
		if edge.Relation != "DEPENDS_ON" {
			t.Errorf("Expected relation 'DEPENDS_ON', got '%s'", edge.Relation)
		}
	}
}

func TestParseGraphWithTerraformStyleLabels(t *testing.T) {
	// Test with real Terraform output style
	dotString := `digraph G {
		rankdir = "RL";
		node [shape = rect, fontname = "sans-serif"];
		"aws_vpc.main" [label="aws_vpc.main"];
		"aws_subnet.public" [label="aws_subnet.public"];
		"aws_subnet.public" -> "aws_vpc.main";
	}`

	graphAst, err := gographviz.ParseString(dotString)
	if err != nil {
		t.Fatalf("Failed to parse DOT string: %v", err)
	}

	dotGraph := gographviz.NewGraph()
	if err := gographviz.Analyse(graphAst, dotGraph); err != nil {
		t.Fatalf("Failed to analyse graph: %v", err)
	}

	g, err := ParseGraph(dotGraph)
	if err != nil {
		t.Fatalf("ParseGraph failed: %v", err)
	}

	// Verify the labels were parsed correctly
	foundVpc := false
	for _, node := range g.Nodes {
		if node.ID == "aws_vpc.main" {
			foundVpc = true
			if node.Type != "aws_vpc" {
				t.Errorf("Expected type 'aws_vpc', got '%s'", node.Type)
			}
			if node.Name != "main" {
				t.Errorf("Expected name 'main', got '%s'", node.Name)
			}
		}
	}
	if !foundVpc {
		t.Error("Did not find aws_vpc.main node")
	}
}

func TestParseGraphNilInput(t *testing.T) {
	_, err := ParseGraph(nil)
	if err == nil {
		t.Error("Expected error for nil input, got nil")
	}
}
