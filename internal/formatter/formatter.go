package formatter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"terraform-graphx/internal/graph"
)

// ToJSON converts a graph object to its JSON string representation.
func ToJSON(g *graph.Graph) (string, error) {
	jsonData, err := json.MarshalIndent(g, "", "  ")
	if err != nil {
		return "", err
	}
	return string(jsonData), nil
}

// ToCypher converts a graph to Cypher MERGE statements for direct execution.
// These statements are idempotent and can be run multiple times safely.
// Use this for simple Cypher script generation.
func ToCypher(g *graph.Graph) (string, error) {
	var sb strings.Builder

	// Create or update all nodes
	for _, node := range g.Nodes {
		sb.WriteString(fmt.Sprintf("MERGE (n:Resource {id: '%s'})\n", escapeString(node.ID)))
		sb.WriteString(fmt.Sprintf("SET n.type = '%s', n.provider = '%s', n.name = '%s';\n",
			escapeString(node.Type),
			escapeString(node.Provider),
			escapeString(node.Name)))
	}

	sb.WriteString("\n")

	// Create or update all relationships
	for _, edge := range g.Edges {
		sb.WriteString(fmt.Sprintf(
			"MATCH (from:Resource {id: '%s'}), (to:Resource {id: '%s'})\n"+
				"MERGE (from)-[:%s]->(to);\n",
			escapeString(edge.From),
			escapeString(edge.To),
			edge.Relation))
	}

	return sb.String(), nil
}

// ToCypherTransaction converts a graph to a parameterized Cypher query.
// This is the recommended approach for Neo4j driver execution as it:
// - Prevents Cypher injection
// - Improves performance through query plan caching
// - Handles special characters automatically
func ToCypherTransaction(g *graph.Graph) (string, map[string]interface{}) {
	var query bytes.Buffer
	params := make(map[string]interface{})

	// Build node data for parameterized query
	nodesData := make([]map[string]interface{}, len(g.Nodes))
	for i, node := range g.Nodes {
		nodesData[i] = map[string]interface{}{
			"id":       node.ID,
			"type":     node.Type,
			"provider": node.Provider,
			"name":     node.Name,
		}
	}
	params["nodes"] = nodesData

	// Create/update nodes using UNWIND for batch processing
	query.WriteString("UNWIND $nodes AS node_data\n")
	query.WriteString("MERGE (n:Resource {id: node_data.id})\n")
	query.WriteString("SET n.type = node_data.type, n.provider = node_data.provider, n.name = node_data.name\n")

	// Build edge data and create relationships if any exist
	if len(g.Edges) > 0 {
		edgesData := make([]map[string]string, len(g.Edges))
		for i, edge := range g.Edges {
			edgesData[i] = map[string]string{
				"from": edge.From,
				"to":   edge.To,
			}
		}
		params["edges"] = edgesData

		query.WriteString("WITH *\n")
		query.WriteString("UNWIND $edges AS edge_data\n")
		query.WriteString("MATCH (from:Resource {id: edge_data.from})\n")
		query.WriteString("MATCH (to:Resource {id: edge_data.to})\n")
		query.WriteString("MERGE (from)-[:DEPENDS_ON]->(to)\n")
	}

	return query.String(), params
}

// escapeString escapes single quotes in Cypher string literals.
func escapeString(s string) string {
	return strings.ReplaceAll(s, "'", "\\'")
}
