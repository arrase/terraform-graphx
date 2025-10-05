package builder

import (
	"fmt"
	"sort"
	"strings"
	"terraform-graphx/internal/graph"
	"terraform-graphx/internal/parser"
)

const (
	ManagedResourceMode = "managed"
	DependsOnRelation   = "DEPENDS_ON"
)

// Build constructs a dependency graph from a parsed Terraform plan.
func Build(plan *parser.TerraformPlan) *graph.Graph {
	g := &graph.Graph{
		Nodes: make([]graph.Node, 0),
		Edges: make([]graph.Edge, 0),
	}

	// Extract all nodes from the plan
	nodes := extractNodes(&plan.PlannedValues.RootModule)
	g.Nodes = append(g.Nodes, nodes...)

	// Build lookup structures for efficient edge extraction
	nodeMap, nodeKeys := createNodeLookupMap(g.Nodes)

	// Extract edges from configuration
	uniqueEdges := make(map[string]struct{})
	extractEdges(&plan.Configuration.RootModule, nodeMap, nodeKeys, uniqueEdges)

	// Convert unique edges map to slice
	g.Edges = convertEdgesToSlice(uniqueEdges)

	return g
}

// createNodeLookupMap creates a map and sorted keys for efficient node lookup.
// Keys are sorted by length (descending) to ensure longest matches are found first.
func createNodeLookupMap(nodes []graph.Node) (map[string]graph.Node, []string) {
	nodeMap := make(map[string]graph.Node)
	nodeKeys := make([]string, 0, len(nodes))

	for _, n := range nodes {
		nodeMap[n.ID] = n
		nodeKeys = append(nodeKeys, n.ID)
	}

	// Sort by length (descending) to match longest addresses first
	// e.g., "module.x.aws_instance.foo" before "module.x"
	sort.Slice(nodeKeys, func(i, j int) bool {
		return len(nodeKeys[i]) > len(nodeKeys[j])
	})

	return nodeMap, nodeKeys
}

// convertEdgesToSlice transforms the unique edges map into a slice of Edge structs.
func convertEdgesToSlice(uniqueEdges map[string]struct{}) []graph.Edge {
	edges := make([]graph.Edge, 0, len(uniqueEdges))

	for edgeKey := range uniqueEdges {
		parts := strings.Split(edgeKey, " -> ")
		edges = append(edges, graph.Edge{
			From:     parts[0],
			To:       parts[1],
			Relation: DependsOnRelation,
		})
	}

	return edges
}

// extractNodes recursively traverses modules to find all managed resources.
func extractNodes(module *parser.Module) []graph.Node {
	var nodes []graph.Node

	for _, r := range module.Resources {
		// Only include managed resources (not data sources)
		if r.Mode == ManagedResourceMode {
			nodes = append(nodes, graph.Node{
				ID:         r.Address,
				Type:       r.Type,
				Provider:   r.ProviderName,
				Name:       r.Name,
				Attributes: r.Values,
			})
		}
	}

	// Recursively process child modules
	for _, child := range module.ChildModules {
		nodes = append(nodes, extractNodes(&child)...)
	}

	return nodes
}

// extractEdges recursively traverses the configuration to find dependencies.
// It populates uniqueEdges map to prevent duplicate edges.
// Dependencies are found by analyzing expressions in resource blocks,
// which contain references to other resources, variables, or data sources.
func extractEdges(module *parser.ConfigModule, nodeMap map[string]graph.Node, nodeKeys []string, uniqueEdges map[string]struct{}) {
	for _, r := range module.Resources {
		// Skip if resource is not in our node map (e.g., data sources)
		if _, ok := nodeMap[r.Address]; !ok {
			continue
		}

		// Process all expressions in the resource
		for _, expr := range r.Expressions {
			for _, ref := range expr.References {
				depAddress := resolveResourceAddress(ref, nodeKeys)

				// Add edge if valid dependency found (no self-references)
				if depAddress != "" && r.Address != depAddress {
					edgeKey := fmt.Sprintf("%s -> %s", r.Address, depAddress)
					uniqueEdges[edgeKey] = struct{}{}
				}
			}
		}
	}

	// Recursively process child modules
	for _, child := range module.ModuleCalls {
		extractEdges(&child.Module, nodeMap, nodeKeys, uniqueEdges)
	}
}

// resolveResourceAddress finds the resource address that matches the given reference.
// Returns empty string if the reference is not a resource (e.g., var, local, or data source).
// It checks for exact matches or attribute references (e.g., resource.attr).
func resolveResourceAddress(ref string, nodeKeys []string) string {
	parts := strings.Split(ref, ".")

	// Filter out non-resource references (variables, locals, etc.)
	if len(parts) > 0 && (parts[0] == "var" || parts[0] == "local") {
		return ""
	}

	// Find the longest matching resource address
	// Check if ref matches exactly or is an attribute/index reference
	for _, key := range nodeKeys {
		if ref == key || strings.HasPrefix(ref, key+".") || strings.HasPrefix(ref, key+"[") {
			return key
		}
	}

	// Handle data sources (not included as nodes but can be dependencies)
	if len(parts) >= 3 && parts[0] == "data" {
		return strings.Join(parts[:3], ".")
	}

	return ""
}
