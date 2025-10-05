package formatter

import (
	"encoding/json"
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