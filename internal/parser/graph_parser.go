package parser

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"terraform-graphx/internal/graph"
)

// DotGraph represents the top-level structure of the JSON output from dot.
type DotGraph struct {
	Objects []DotObject `json:"objects"`
	Edges   []DotEdge   `json:"edges"`
}

// DotObject represents a node or a subgraph in the dot output.
type DotObject struct {
	Name  string `json:"name"`
	Label string `json:"label"`
	// We only care about nodes, which have a _gvid
	GVID *int `json:"_gvid"`
}

// DotEdge represents an edge in the dot output.
type DotEdge struct {
	Tail int `json:"tail"`
	Head int `json:"head"`
}

// cleanLabel removes extra quoting and formatting from the dot label.
func cleanLabel(label string) string {
	re := regexp.MustCompile(`\["(.*?)"\]`)
	matches := re.FindStringSubmatch(label)
	if len(matches) > 1 {
		return matches[1]
	}
	return label
}

// ParseGraph parses the JSON output from `dot -Tjson`.
func ParseGraph(data []byte) (*graph.Graph, error) {
	var dotGraph DotGraph
	if err := json.Unmarshal(data, &dotGraph); err != nil {
		return nil, fmt.Errorf("failed to unmarshal dot json: %w", err)
	}

	g := &graph.Graph{
		Nodes: make([]graph.Node, 0),
		Edges: make([]graph.Edge, 0),
	}

	nodeMap := make(map[int]string)

	// Extract nodes
	for _, obj := range dotGraph.Objects {
		// We identify nodes by checking for the _gvid attribute, which is absent in subgraphs.
		if obj.GVID != nil {
			// Clean up the label to get the resource address
			address := cleanLabel(obj.Name)
			nodeMap[*obj.GVID] = address

			// Basic type/name extraction
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
				Provider: "", // This information is not in the graph output
			})
		}
	}

	// Extract edges
	for _, edge := range dotGraph.Edges {
		from, okFrom := nodeMap[edge.Tail]
		to, okTo := nodeMap[edge.Head]

		if okFrom && okTo {
			g.Edges = append(g.Edges, graph.Edge{
				From:     from,
				To:       to,
				Relation: "DEPENDS_ON",
			})
		}
	}

	return g, nil
}
