package neo4j

import (
	"context"
	"fmt"
	"terraform-graphx/internal/formatter"
	"terraform-graphx/internal/graph"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// Client handles the connection and communication with a Neo4j database.
type Client struct {
	Driver neo4j.DriverWithContext
	URI    string
	User   string
	Pass   string
}

// NewClient creates a new Neo4j client and establishes a connection.
func NewClient(uri, user, pass string) (*Client, error) {
	driver, err := neo4j.NewDriverWithContext(uri, neo4j.BasicAuth(user, pass, ""))
	if err != nil {
		return nil, fmt.Errorf("could not create neo4j driver: %w", err)
	}

	return &Client{
		Driver: driver,
		URI:    uri,
		User:   user,
		Pass:   pass,
	}, nil
}

// Close gracefully shuts down the driver.
func (c *Client) Close(ctx context.Context) error {
	return c.Driver.Close(ctx)
}

// VerifyConnectivity checks if a connection can be established with the database.
func (c *Client) VerifyConnectivity(ctx context.Context) error {
	return c.Driver.VerifyConnectivity(ctx)
}

// UpdateGraph synchronizes the Neo4j database with the current graph state.
// It removes obsolete resources and relationships, then upserts the current ones.
func (c *Client) UpdateGraph(ctx context.Context, g *graph.Graph) error {
	session := c.Driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close(ctx)

	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		// First, get all current resource IDs from Neo4j
		currentResourcesQuery := "MATCH (n:Resource) RETURN n.id as id"
		result, err := tx.Run(ctx, currentResourcesQuery, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to query current resources: %w", err)
		}

		// Collect current resource IDs
		existingIDs := make(map[string]bool)
		for result.Next(ctx) {
			record := result.Record()
			if id, ok := record.Get("id"); ok {
				if idStr, ok := id.(string); ok {
					existingIDs[idStr] = true
				}
			}
		}
		if err := result.Err(); err != nil {
			return nil, fmt.Errorf("failed to iterate current resources: %w", err)
		}

		// Build set of new resource IDs
		newIDs := make(map[string]bool)
		for _, node := range g.Nodes {
			newIDs[node.ID] = true
		}

		// Find resources to delete (exist in Neo4j but not in new graph)
		var idsToDelete []string
		for existingID := range existingIDs {
			if !newIDs[existingID] {
				idsToDelete = append(idsToDelete, existingID)
			}
		}

		// Delete obsolete resources and their relationships
		if len(idsToDelete) > 0 {
			deleteQuery := "UNWIND $obsoleteIds AS obsoleteId MATCH (n:Resource {id: obsoleteId}) DETACH DELETE n"
			deleteParams := map[string]interface{}{
				"obsoleteIds": idsToDelete,
			}
			_, err := tx.Run(ctx, deleteQuery, deleteParams)
			if err != nil {
				return nil, fmt.Errorf("failed to delete obsolete resources: %w", err)
			}
		}

		// Now upsert the current graph
		query, params := formatter.ToCypherTransaction(g)
		result, err = tx.Run(ctx, query, params)
		if err != nil {
			return nil, fmt.Errorf("failed to upsert current graph: %w", err)
		}
		return result.Consume(ctx)
	})

	if err != nil {
		return fmt.Errorf("failed to execute transaction to update graph: %w", err)
	}

	return nil
}
