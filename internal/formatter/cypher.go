package formatter

import (
	"bytes"
	"fmt"
	"strings"
	"terraform-graphx/internal/graph"
)

// ToCypher converts a graph object to a series of idempotent Cypher MERGE statements.
func ToCypher(g *graph.Graph) (string, error) {
	var sb strings.Builder

	// Generate MERGE statements for nodes
	for _, node := range g.Nodes {
		// Using MERGE to ensure idempotency. It will match existing nodes on 'id' or create them.
		sb.WriteString(fmt.Sprintf("MERGE (n:Resource {id: '%s'})\n", node.ID))
		// Use SET to add or update properties. This is cleaner than including them in the MERGE.
		sb.WriteString(fmt.Sprintf("SET n.type = '%s', n.provider = '%s', n.name = '%s';\n", node.Type, node.Provider, node.Name))
	}

	sb.WriteString("\n")

	// Generate MERGE statements for edges
	for _, edge := range g.Edges {
		// MERGE the relationship between the two nodes.
		// This assumes the nodes have already been created by the statements above.
		cypher := fmt.Sprintf(
			"MATCH (from:Resource {id: '%s'}), (to:Resource {id: '%s'})\nMERGE (from)-[:%s]->(to);\n",
			edge.From,
			edge.To,
			edge.Relation,
		)
		sb.WriteString(cypher)
	}

	return sb.String(), nil
}

// ToCypherTransaction converts a graph into a single transaction with parameters.
// This is a more robust way to interact with Neo4j.
func ToCypherTransaction(g *graph.Graph) (string, map[string]interface{}) {
	var query bytes.Buffer
	params := make(map[string]interface{})

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
	query.WriteString("UNWIND $nodes AS node_data\n")
	query.WriteString("MERGE (n:Resource {id: node_data.id})\n")
	query.WriteString("SET n.type = node_data.type, n.provider = node_data.provider, n.name = node_data.name\n")

	if len(g.Edges) > 0 {
		edgesData := make([]map[string]string, len(g.Edges))
		for i, edge := range g.Edges {
			edgesData[i] = map[string]string{
				"from": edge.From,
				"to":   edge.To,
			}
		}
		params["edges"] = edgesData
		query.WriteString("WITH * \n")
		query.WriteString("UNWIND $edges AS edge_data\n")
		query.WriteString("MATCH (from:Resource {id: edge_data.from})\n")
		query.WriteString("MATCH (to:Resource {id: edge_data.to})\n")
		query.WriteString("MERGE (from)-[:DEPENDS_ON]->(to)\n")
	}

	return query.String(), params
}