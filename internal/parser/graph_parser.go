package parser

import (
	"fmt"
	"regexp"
	"strings"
	"terraform-graphx/internal/graph"

	"github.com/awalterschulze/gographviz"
)

// cleanLabel removes extra quoting and formatting from node labels.
func cleanLabel(label string) string {
	// Remove surrounding quotes if present
	label = strings.Trim(label, `"`)

	// Handle Terraform-style labels like ["resource.name"]
	re := regexp.MustCompile(`\["(.*?)"\]`)
	matches := re.FindStringSubmatch(label)
	if len(matches) > 1 {
		return matches[1]
	}
	return label
}

// ParseGraph converts a gographviz.Graph directly to our internal graph structure.
// This eliminates the need for an intermediate JSON conversion step.
func ParseGraph(dotGraph *gographviz.Graph) (*graph.Graph, error) {
	if dotGraph == nil {
		return nil, fmt.Errorf("dotGraph cannot be nil")
	}

	g := &graph.Graph{
		Nodes: make([]graph.Node, 0),
		Edges: make([]graph.Edge, 0),
	}

	nodeMap := make(map[string]string) // maps original node name -> cleaned address

	// Extract nodes from gographviz
	for nodeName, node := range dotGraph.Nodes.Lookup {
		// Get the label if it exists, otherwise use the node name
		label := nodeName
		if node.Attrs != nil {
			if labelAttr, ok := node.Attrs["label"]; ok {
				label = labelAttr
			}
		}

		// Clean up the label to get the resource address
		address := cleanLabel(label)
		nodeMap[nodeName] = address

		// Extract type and name from the address
		// Example: "aws_instance.web" -> type="aws_instance", name="web"
		parts := strings.Split(address, ".")
		var nodeType, nodeName string
		if len(parts) >= 2 {
			nodeType = parts[len(parts)-2]
			nodeName = parts[len(parts)-1]
		}

		g.Nodes = append(g.Nodes, graph.Node{
			ID:       address,
			Type:     nodeType,
			Name:     nodeName,
			Provider: "", // Provider info is not available in the graph output
		})
	}

	// Extract edges from gographviz
	for _, edge := range dotGraph.Edges.Edges {
		fromAddr, okFrom := nodeMap[edge.Src]
		toAddr, okTo := nodeMap[edge.Dst]

		if okFrom && okTo {
			g.Edges = append(g.Edges, graph.Edge{
				From:     fromAddr,
				To:       toAddr,
				Relation: "DEPENDS_ON",
			})
		}
	}

	return g, nil
}
