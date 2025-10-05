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
}

// NewClient creates a new Neo4j client and establishes a connection.
func NewClient(uri, user, pass string) (*Client, error) {
	driver, err := neo4j.NewDriverWithContext(uri, neo4j.BasicAuth(user, pass, ""))
	if err != nil {
		return nil, fmt.Errorf("could not create neo4j driver: %w", err)
	}

	return &Client{Driver: driver}, nil
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
		// Get current state from Neo4j
		existingIDs, err := c.fetchExistingResourceIDs(ctx, tx)
		if err != nil {
			return nil, err
		}

		// Remove obsolete resources
		if err := c.deleteObsoleteResources(ctx, tx, existingIDs, g); err != nil {
			return nil, err
		}

		// Upsert current graph state
		return c.upsertGraph(ctx, tx, g)
	})

	if err != nil {
		return fmt.Errorf("failed to update graph: %w", err)
	}

	return nil
}

// fetchExistingResourceIDs retrieves all resource IDs currently in Neo4j.
func (c *Client) fetchExistingResourceIDs(ctx context.Context, tx neo4j.ManagedTransaction) (map[string]bool, error) {
	query := "MATCH (n:Resource) RETURN n.id as id"
	result, err := tx.Run(ctx, query, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to query existing resources: %w", err)
	}

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
		return nil, fmt.Errorf("failed to iterate existing resources: %w", err)
	}

	return existingIDs, nil
}

// deleteObsoleteResources removes resources that exist in Neo4j but not in the new graph.
func (c *Client) deleteObsoleteResources(ctx context.Context, tx neo4j.ManagedTransaction, existingIDs map[string]bool, g *graph.Graph) error {
	// Build set of new resource IDs
	newIDs := make(map[string]bool, len(g.Nodes))
	for _, node := range g.Nodes {
		newIDs[node.ID] = true
	}

	// Find resources to delete
	var idsToDelete []string
	for existingID := range existingIDs {
		if !newIDs[existingID] {
			idsToDelete = append(idsToDelete, existingID)
		}
	}

	// Delete obsolete resources and their relationships
	if len(idsToDelete) > 0 {
		query := "UNWIND $obsoleteIds AS obsoleteId MATCH (n:Resource {id: obsoleteId}) DETACH DELETE n"
		params := map[string]interface{}{"obsoleteIds": idsToDelete}

		if _, err := tx.Run(ctx, query, params); err != nil {
			return fmt.Errorf("failed to delete obsolete resources: %w", err)
		}
	}

	return nil
}

// upsertGraph inserts or updates the current graph state in Neo4j.
func (c *Client) upsertGraph(ctx context.Context, tx neo4j.ManagedTransaction, g *graph.Graph) (interface{}, error) {
	query, params := formatter.ToCypherTransaction(g)
	result, err := tx.Run(ctx, query, params)
	if err != nil {
		return nil, fmt.Errorf("failed to upsert graph: %w", err)
	}
	return result.Consume(ctx)
}
