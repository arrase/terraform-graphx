package formatter

import (
	"bytes"
	"terraform-graphx/internal/graph"
)

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
