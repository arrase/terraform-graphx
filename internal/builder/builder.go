package builder

import (
	"fmt"
	"sort"
	"strings"
	"terraform-graphx/internal/graph"
	"terraform-graphx/internal/parser"
)

// Build constructs a graph from a parsed Terraform plan.
func Build(plan *parser.TerraformPlan) *graph.Graph {
	g := &graph.Graph{
		Nodes: make([]graph.Node, 0),
		Edges: make([]graph.Edge, 0),
	}

	nodes := extractNodes(&plan.PlannedValues.RootModule)
	g.Nodes = append(g.Nodes, nodes...)

	// Create a map for quick node lookup by address
	nodeMap := make(map[string]graph.Node)
	// Also create a sorted list of keys to ensure longest match is found correctly.
	var nodeKeys []string
	for _, n := range g.Nodes {
		nodeMap[n.ID] = n
		nodeKeys = append(nodeKeys, n.ID)
	}

	// Sort keys by length in descending order. This ensures we match the longest
	// possible resource address first (e.g., "module.x.aws_instance.foo" before "module.x").
	sort.Slice(nodeKeys, func(i, j int) bool {
		return len(nodeKeys[i]) > len(nodeKeys[j])
	})

	// Use a map to store unique edges to prevent duplicates
	uniqueEdges := make(map[string]struct{})
	extractEdges(&plan.Configuration.RootModule, nodeMap, nodeKeys, uniqueEdges)

	// Convert the map of unique edges into a slice of Edge structs
	for edgeKey := range uniqueEdges {
		parts := strings.Split(edgeKey, " -> ")
		g.Edges = append(g.Edges, graph.Edge{
			From:     parts[0],
			To:       parts[1],
			Relation: "DEPENDS_ON",
		})
	}

	return g
}

// extractNodes recursively traverses the modules to find all resources.
func extractNodes(module *parser.Module) []graph.Node {
	var nodes []graph.Node

	for _, r := range module.Resources {
		// We only care about managed resources, not data sources, for the primary graph nodes.
		if r.Mode == "managed" {
			nodes = append(nodes, graph.Node{
				ID:         r.Address,
				Type:       r.Type,
				Provider:   r.ProviderName,
				Name:       r.Name,
				Attributes: r.Values,
			})
		}
	}

	for _, child := range module.ChildModules {
		nodes = append(nodes, extractNodes(&child)...)
	}

	return nodes
}

// extractEdges recursively traverses the configuration to find dependencies, populating a map to ensure uniqueness.
func extractEdges(module *parser.ConfigModule, nodeMap map[string]graph.Node, nodeKeys []string, uniqueEdges map[string]struct{}) {
	for _, r := range module.Resources {
		// The resource might not be in our node map if it's a data source, which is fine.
		// We only care about the origin being a managed resource.
		if _, ok := nodeMap[r.Address]; !ok {
			continue
		}

		for _, expr := range r.Expressions {
			for _, ref := range expr.References {
				depAddress := getRootResourceAddress(ref, nodeKeys)
				if depAddress != "" {
					// Ensure we don't create self-dependencies
					if r.Address != depAddress {
						// Create a unique key for the edge and add it to the map
						edgeKey := fmt.Sprintf("%s -> %s", r.Address, depAddress)
						uniqueEdges[edgeKey] = struct{}{}
					}
				}
			}
		}
	}

	for _, child := range module.ModuleCalls {
		extractEdges(&child.Module, nodeMap, nodeKeys, uniqueEdges)
	}
}

// getRootResourceAddress finds the longest resource address that is a prefix of the reference.
func getRootResourceAddress(ref string, nodeKeys []string) string {
	// Filter out non-resource references like variables or locals.
	parts := strings.Split(ref, ".")
	if len(parts) > 0 && (parts[0] == "var" || parts[0] == "local") {
		return ""
	}

	// Iterate through the sorted keys (longest to shortest) to find the first match.
	for _, key := range nodeKeys {
		// Check if 'ref' is exactly the key or if it's a reference to an attribute (e.g., key.id or key[0].id)
		if ref == key || strings.HasPrefix(ref, key+".") || strings.HasPrefix(ref, key+"[") {
			return key
		}
	}

	// It might be a data source, which we don't include as nodes, but they can be dependencies.
	if len(parts) > 0 && parts[0] == "data" {
		// Return the data source address so it can be handled if needed in the future.
		return strings.Join(parts[:3], ".")
	}

	return ""
}