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

// UpdateGraph upserts the graph data into the Neo4j database within a single transaction.
func (c *Client) UpdateGraph(ctx context.Context, g *graph.Graph) error {
	session := c.Driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close(ctx)

	query, params := formatter.ToCypherTransaction(g)

	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		result, err := tx.Run(ctx, query, params)
		if err != nil {
			return nil, err
		}
		return result.Consume(ctx)
	})

	if err != nil {
		return fmt.Errorf("failed to execute transaction to update graph: %w", err)
	}

	return nil
}