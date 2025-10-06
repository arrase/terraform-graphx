package builder

import (
	"bytes"
	"encoding/json"
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

	// Extract edges from the prior state's depends_on fields
	uniqueEdges := make(map[string]struct{})
	extractEdgesFromState(&plan.PriorState.Values.RootModule, nodeMap, uniqueEdges)

	// Fallback to configuration analysis if state is empty (e.g., initial plan)
	if len(uniqueEdges) == 0 {
		extractEdgesFromConfig(&plan.Configuration.RootModule, nodeMap, nodeKeys, uniqueEdges)
	}

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

// extractEdgesFromState recursively traverses the state to find dependencies from `depends_on`.
func extractEdgesFromState(module *parser.StateModule, nodeMap map[string]graph.Node, uniqueEdges map[string]struct{}) {
	for _, r := range module.Resources {
		// Ensure the resource exists in our node map
		if _, ok := nodeMap[r.Address]; !ok {
			continue
		}

		for _, dep := range r.DependsOn {
			// The dep string is already a resolved resource address
			if _, ok := nodeMap[dep]; ok {
				edgeKey := fmt.Sprintf("%s -> %s", r.Address, dep)
				uniqueEdges[edgeKey] = struct{}{}
			}
		}
	}

	// Recursively process child modules
	for _, child := range module.ChildModules {
		extractEdgesFromState(&child, nodeMap, uniqueEdges)
	}
}

// extractEdgesFromConfig recursively traverses the configuration to find dependencies.
func extractEdgesFromConfig(module *parser.ConfigModule, nodeMap map[string]graph.Node, nodeKeys []string, uniqueEdges map[string]struct{}) {
	for _, r := range module.Resources {
		var resource parser.ConfigResource
		if err := json.Unmarshal(r, &resource); err != nil {
			continue
		}

		// Skip if resource is not in our node map (e.g., data sources)
		if _, ok := nodeMap[resource.Address]; !ok {
			continue
		}

		// Process all expressions in the resource
		findReferencesInRawMessage(resource.Expressions, func(ref string) {
			depAddress := resolveResourceAddress(ref, nodeKeys)
			// Add edge if valid dependency found (no self-references)
			if depAddress != "" && resource.Address != depAddress {
				edgeKey := fmt.Sprintf("%s -> %s", resource.Address, depAddress)
				uniqueEdges[edgeKey] = struct{}{}
			}
		})
	}

	// Recursively process child modules
	for _, child := range module.ModuleCalls {
		extractEdgesFromConfig(&child.Module, nodeMap, nodeKeys, uniqueEdges)
	}
}

// findReferencesInRawMessage recursively decodes a json.RawMessage to find all "references" fields.
func findReferencesInRawMessage(raw json.RawMessage, callback func(string)) {
	if len(raw) == 0 || bytes.Equal(raw, []byte("null")) {
		return
	}

	// Case 1: It's an object
	if bytes.HasPrefix(raw, []byte("{")) {
		var obj map[string]json.RawMessage
		if err := json.Unmarshal(raw, &obj); err != nil {
			return
		}

		// Check if this object is an Expression
		if refs, ok := obj["references"]; ok {
			var references []string
			if err := json.Unmarshal(refs, &references); err == nil {
				for _, ref := range references {
					callback(ref)
				}
			}
		} else {
			// Otherwise, recurse into its values
			for _, value := range obj {
				findReferencesInRawMessage(value, callback)
			}
		}
		return
	}

	// Case 2: It's an array
	if bytes.HasPrefix(raw, []byte("[")) {
		var arr []json.RawMessage
		if err := json.Unmarshal(raw, &arr); err != nil {
			return
		}
		for _, item := range arr {
			findReferencesInRawMessage(item, callback)
		}
		return
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
