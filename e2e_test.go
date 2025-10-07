package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"terraform-graphx/internal/config"
	"terraform-graphx/internal/neo4j"

	neo4jdriver "github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

const (
	e2eTimeout = 60 * time.Second
)

// getBinaryPath returns the absolute path to the terraform-graphx binary
func getBinaryPath() string {
	cwd, _ := os.Getwd()
	return filepath.Join(cwd, "terraform-graphx")
}

// TestE2E_FullWorkflow tests the complete end-to-end workflow
func TestE2E_FullWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	// Load config
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if cfg.Neo4j.Password == "" {
		t.Skip("Neo4j password not configured in .terraform-graphx.yaml, skipping E2E test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), e2eTimeout)
	defer cancel()

	// Verify Neo4j connectivity first
	client, err := neo4j.NewClient(cfg.Neo4j.URI, cfg.Neo4j.User, cfg.Neo4j.Password)
	if err != nil {
		t.Fatalf("Failed to create Neo4j client: %v", err)
	}
	defer client.Close(ctx)

	if err := client.VerifyConnectivity(ctx); err != nil {
		t.Skipf("Cannot connect to Neo4j at %s: %v", cfg.Neo4j.URI, err)
	}

	t.Log("✓ Connected to Neo4j successfully")

	// Test 1: Clear database
	t.Run("1_ClearDatabase", func(t *testing.T) {
		clearNeo4jDatabase(t, ctx, client)
		t.Log("✓ Database cleared")
	})

	// Test 2: Verify examples directory and terraform setup
	t.Run("2_VerifyTerraformSetup", func(t *testing.T) {
		examplesDir := filepath.Join(".", "examples")
		if _, err := os.Stat(examplesDir); os.IsNotExist(err) {
			t.Fatalf("Examples directory not found: %s", examplesDir)
		}

		// Check if terraform is initialized
		if _, err := os.Stat(filepath.Join(examplesDir, ".terraform")); os.IsNotExist(err) {
			t.Log("Terraform not initialized, running terraform init...")
			cmd := exec.Command("terraform", "init")
			cmd.Dir = examplesDir
			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("terraform init failed: %v\nOutput: %s", err, output)
			}
		}

		t.Log("✓ Terraform setup verified")
	})

	// Test 3: Insert data into Neo4j
	t.Run("3_InsertIntoNeo4j", func(t *testing.T) {
		examplesDir := filepath.Join(".", "examples")

		cmd := exec.Command(getBinaryPath(), "update")
		cmd.Dir = examplesDir

		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("graphx update failed: %v\nOutput: %s", err, output)
		}

		outputStr := string(output)
		if !strings.Contains(outputStr, "Successfully updated Neo4j database") {
			t.Errorf("Expected success message in output: %s", outputStr)
		}

		t.Log("✓ Data inserted into Neo4j")
		t.Logf("Output: %s", outputStr)
	})

	// Test 4: Verify nodes in database
	t.Run("4_VerifyNodesInDatabase", func(t *testing.T) {
		count := countNodesInNeo4j(t, ctx, client)

		if count < 2 {
			t.Errorf("Expected at least 2 nodes in database, got %d", count)
		}

		t.Logf("✓ Found %d nodes in Neo4j", count)
	})

	// Test 5: Verify specific resources
	t.Run("5_VerifySpecificResources", func(t *testing.T) {
		expectedResources := map[string]map[string]string{
			"null_resource.cluster": {
				"type": "null_resource",
				"name": "cluster",
			},
			"null_resource.app": {
				"type": "null_resource",
				"name": "app",
			},
		}

		for resourceID, expectedAttrs := range expectedResources {
			attrs := getResourceFromNeo4j(t, ctx, client, resourceID)

			if attrs == nil {
				t.Errorf("Resource %s not found in database", resourceID)
				continue
			}

			for key, expectedValue := range expectedAttrs {
				if actualValue, ok := attrs[key].(string); !ok || actualValue != expectedValue {
					t.Errorf("Resource %s: expected %s=%s, got %v",
						resourceID, key, expectedValue, attrs[key])
				}
			}

			t.Logf("✓ Verified resource %s: %v", resourceID, attrs)
		}
	})

	// Test 6: Verify relationships
	t.Run("6_VerifyRelationships", func(t *testing.T) {
		count := countRelationshipsInNeo4j(t, ctx, client)

		if count < 1 {
			t.Errorf("Expected at least 1 DEPENDS_ON relationship, got %d", count)
		}

		t.Logf("✓ Found %d DEPENDS_ON relationships in Neo4j", count)

		// Verify specific dependency: app -> cluster
		hasDependency := verifyDependency(t, ctx, client, "null_resource.app", "null_resource.cluster")
		if !hasDependency {
			t.Error("Expected dependency from null_resource.app to null_resource.cluster")
		} else {
			t.Log("✓ Verified dependency: null_resource.app → null_resource.cluster")
		}
	})

	// Test 7: Test idempotency (update again)
	t.Run("7_TestIdempotency", func(t *testing.T) {
		examplesDir := filepath.Join(".", "examples")

		// Get count before second update
		countBefore := countNodesInNeo4j(t, ctx, client)

		// Update again
		cmd := exec.Command(getBinaryPath(), "update")
		cmd.Dir = examplesDir

		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Second update failed: %v\nOutput: %s", err, output)
		}

		// Get count after second update
		countAfter := countNodesInNeo4j(t, ctx, client)

		if countBefore != countAfter {
			t.Errorf("Idempotency check failed: node count changed from %d to %d",
				countBefore, countAfter)
		}

		t.Logf("✓ Idempotency verified: %d nodes before and after second update", countAfter)
	})

	// Test 8: Query the graph
	t.Run("8_QueryGraph", func(t *testing.T) {
		// Test a more complex query
		session := client.Driver.NewSession(ctx, neo4jdriver.SessionConfig{AccessMode: neo4jdriver.AccessModeRead})
		defer session.Close(ctx)

		result, err := session.ExecuteRead(ctx, func(tx neo4jdriver.ManagedTransaction) (interface{}, error) {
			// Find all resources that have dependencies
			query := `
				MATCH (source:Resource)-[:DEPENDS_ON]->(target:Resource)
				RETURN source.id as source_id, target.id as target_id, 
				       source.type as source_type, target.type as target_type
			`
			res, err := tx.Run(ctx, query, nil)
			if err != nil {
				return nil, err
			}

			var dependencies []map[string]string
			for res.Next(ctx) {
				record := res.Record()
				dep := make(map[string]string)
				if v, ok := record.Get("source_id"); ok {
					dep["source_id"] = v.(string)
				}
				if v, ok := record.Get("target_id"); ok {
					dep["target_id"] = v.(string)
				}
				if v, ok := record.Get("source_type"); ok {
					dep["source_type"] = v.(string)
				}
				if v, ok := record.Get("target_type"); ok {
					dep["target_type"] = v.(string)
				}
				dependencies = append(dependencies, dep)
			}

			return dependencies, res.Err()
		})

		if err != nil {
			t.Fatalf("Failed to query dependencies: %v", err)
		}

		deps := result.([]map[string]string)
		t.Logf("✓ Found %d dependencies:", len(deps))
		for _, dep := range deps {
			t.Logf("  %s (%s) → %s (%s)",
				dep["source_id"], dep["source_type"],
				dep["target_id"], dep["target_type"])
		}
	})
}

// TestE2E_ConfigCommands tests configuration management commands
func TestE2E_ConfigCommands(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	tmpDir := t.TempDir()

	t.Run("InitConfig", func(t *testing.T) {
		cmd := exec.Command(getBinaryPath(), "init", "config")
		cmd.Dir = tmpDir

		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("init config failed: %v\nOutput: %s", err, output)
		}

		configPath := filepath.Join(tmpDir, ".terraform-graphx.yaml")
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			t.Fatalf("Config file was not created at %s", configPath)
		}

		t.Log("✓ Config file created successfully")
	})

	t.Run("InitConfig_AlreadyExists", func(t *testing.T) {
		cmd := exec.Command(getBinaryPath(), "init", "config")
		cmd.Dir = tmpDir

		output, err := cmd.CombinedOutput()
		if err == nil {
			t.Fatal("Expected error when config already exists")
		}

		if !strings.Contains(string(output), "already exists") {
			t.Errorf("Expected 'already exists' error, got: %s", output)
		}

		t.Log("✓ Correctly rejected duplicate config")
	})
}

// TestE2E_CheckDatabase tests database connectivity check
func TestE2E_CheckDatabase(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if cfg.Neo4j.Password == "" {
		t.Skip("Neo4j password not configured, skipping check database test")
	}

	t.Run("CheckDatabase_Success", func(t *testing.T) {
		cmd := exec.Command(getBinaryPath(), "check", "database")
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("check database failed: %v\nOutput: %s", err, output)
		}

		outputStr := string(output)
		if !strings.Contains(outputStr, "Successfully connected") {
			t.Errorf("Expected success message, got: %s", outputStr)
		}

		t.Log("✓ Database connectivity check passed")
		t.Logf("Output: %s", outputStr)
	})
}

// Helper functions

func clearNeo4jDatabase(t *testing.T, ctx context.Context, client *neo4j.Client) {
	session := client.Driver.NewSession(ctx, neo4jdriver.SessionConfig{AccessMode: neo4jdriver.AccessModeWrite})
	defer session.Close(ctx)

	_, err := session.ExecuteWrite(ctx, func(tx neo4jdriver.ManagedTransaction) (interface{}, error) {
		_, err := tx.Run(ctx, "MATCH (n:Resource) DETACH DELETE n", nil)
		return nil, err
	})

	if err != nil {
		t.Fatalf("Failed to clear database: %v", err)
	}
}

func countNodesInNeo4j(t *testing.T, ctx context.Context, client *neo4j.Client) int64 {
	session := client.Driver.NewSession(ctx, neo4jdriver.SessionConfig{AccessMode: neo4jdriver.AccessModeRead})
	defer session.Close(ctx)

	result, err := session.ExecuteRead(ctx, func(tx neo4jdriver.ManagedTransaction) (interface{}, error) {
		res, err := tx.Run(ctx, "MATCH (n:Resource) RETURN count(n) as count", nil)
		if err != nil {
			return int64(0), err
		}

		if res.Next(ctx) {
			record := res.Record()
			count, _ := record.Get("count")
			return count.(int64), nil
		}

		return int64(0), fmt.Errorf("no result returned")
	})

	if err != nil {
		t.Fatalf("Failed to count nodes: %v", err)
	}

	return result.(int64)
}

func countRelationshipsInNeo4j(t *testing.T, ctx context.Context, client *neo4j.Client) int64 {
	session := client.Driver.NewSession(ctx, neo4jdriver.SessionConfig{AccessMode: neo4jdriver.AccessModeRead})
	defer session.Close(ctx)

	result, err := session.ExecuteRead(ctx, func(tx neo4jdriver.ManagedTransaction) (interface{}, error) {
		res, err := tx.Run(ctx, "MATCH ()-[r:DEPENDS_ON]->() RETURN count(r) as count", nil)
		if err != nil {
			return int64(0), err
		}

		if res.Next(ctx) {
			record := res.Record()
			count, _ := record.Get("count")
			return count.(int64), nil
		}

		return int64(0), fmt.Errorf("no result returned")
	})

	if err != nil {
		t.Fatalf("Failed to count relationships: %v", err)
	}

	return result.(int64)
}

func getResourceFromNeo4j(t *testing.T, ctx context.Context, client *neo4j.Client, resourceID string) map[string]interface{} {
	session := client.Driver.NewSession(ctx, neo4jdriver.SessionConfig{AccessMode: neo4jdriver.AccessModeRead})
	defer session.Close(ctx)

	result, err := session.ExecuteRead(ctx, func(tx neo4jdriver.ManagedTransaction) (interface{}, error) {
		query := "MATCH (n:Resource {id: $id}) RETURN n.id as id, n.type as type, n.name as name, n.provider as provider"
		res, err := tx.Run(ctx, query, map[string]interface{}{"id": resourceID})
		if err != nil {
			return nil, err
		}

		if res.Next(ctx) {
			record := res.Record()
			attrs := make(map[string]interface{})

			if v, ok := record.Get("id"); ok {
				attrs["id"] = v
			}
			if v, ok := record.Get("type"); ok {
				attrs["type"] = v
			}
			if v, ok := record.Get("name"); ok {
				attrs["name"] = v
			}
			if v, ok := record.Get("provider"); ok {
				attrs["provider"] = v
			}

			return attrs, nil
		}

		return nil, nil
	})

	if err != nil {
		t.Fatalf("Failed to get resource %s: %v", resourceID, err)
	}

	if result == nil {
		return nil
	}

	return result.(map[string]interface{})
}

func verifyDependency(t *testing.T, ctx context.Context, client *neo4j.Client, sourceID, targetID string) bool {
	session := client.Driver.NewSession(ctx, neo4jdriver.SessionConfig{AccessMode: neo4jdriver.AccessModeRead})
	defer session.Close(ctx)

	result, err := session.ExecuteRead(ctx, func(tx neo4jdriver.ManagedTransaction) (interface{}, error) {
		query := `
			MATCH (source:Resource {id: $sourceID})-[:DEPENDS_ON]->(target:Resource {id: $targetID})
			RETURN count(*) as count
		`
		res, err := tx.Run(ctx, query, map[string]interface{}{
			"sourceID": sourceID,
			"targetID": targetID,
		})
		if err != nil {
			return int64(0), err
		}

		if res.Next(ctx) {
			record := res.Record()
			count, _ := record.Get("count")
			return count.(int64), nil
		}

		return int64(0), fmt.Errorf("no result returned")
	})

	if err != nil {
		t.Fatalf("Failed to verify dependency: %v", err)
	}

	return result.(int64) > 0
}
