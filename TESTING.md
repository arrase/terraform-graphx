# Testing terraform-graphx

This document describes how to run the test suite for terraform-graphx.

## Test Types

### Unit Tests

Unit tests test individual components in isolation without external dependencies:

```bash
make test-unit
# or
go test -v -short ./...
```

These tests run quickly and don't require Neo4j or any external services.

### End-to-End (E2E) Tests

End-to-end tests verify the complete workflow from Terraform resources to Neo4j database:

```bash
make test-e2e
```

## E2E Test Setup

### Prerequisites

1. **Neo4j running** - You need a Neo4j instance running and accessible
2. **Configuration file** - A `.terraform-graphx.yaml` file with valid credentials

### Setting up Neo4j for Testing

#### Option 1: Docker (Recommended)

```bash
# Start Neo4j in Docker
docker run -d \
  --name terraform-graphx-neo4j-test \
  -p 7474:7474 -p 7687:7687 \
  -v $(pwd)/neo4j-data:/data \
  -e NEO4J_AUTH=neo4j/testpassword \
  neo4j:community

# Wait a few seconds for Neo4j to start
sleep 10
```

#### Option 2: Existing Neo4j Instance

Use any Neo4j instance you have available (Community or Enterprise Edition).

### Configure Credentials

1. **Initialize the config file:**

```bash
./terraform-graphx init config
```

2. **Edit `.terraform-graphx.yaml`:**

```yaml
neo4j:
  uri: bolt://localhost:7687
  user: neo4j
  password: testpassword  # Use your actual password
```

## Running E2E Tests

### Quick Start

```bash
# Build and run all E2E tests
make test-e2e
```

### What the E2E Tests Verify

The complete test suite (`TestE2E_FullWorkflow`) performs the following checks:

1. **✓ Database Connectivity** - Verifies connection to Neo4j
2. **✓ Clear Database** - Cleans any existing test data
3. **✓ Terraform Setup** - Ensures Terraform is initialized in examples directory
4. **✓ JSON Output Generation** - Generates and validates JSON graph format
5. **✓ Cypher Output Generation** - Generates and validates Cypher statements
6. **✓ Data Insertion** - Inserts Terraform resources into Neo4j
7. **✓ Node Verification** - Confirms nodes exist in database with correct attributes
8. **✓ Relationship Verification** - Validates DEPENDS_ON relationships
9. **✓ Idempotency** - Ensures running the same command twice doesn't duplicate data
10. **✓ Graph Queries** - Tests complex Cypher queries on the data

### Example Test Output

```
=== RUN   TestE2E_FullWorkflow
    e2e_test.go:60: ✓ Connected to Neo4j successfully
=== RUN   TestE2E_FullWorkflow/1_ClearDatabase
    e2e_test.go:65: ✓ Database cleared
=== RUN   TestE2E_FullWorkflow/2_VerifyTerraformSetup
    e2e_test.go:86: ✓ Terraform setup verified
=== RUN   TestE2E_FullWorkflow/3_GenerateJSONOutput
    e2e_test.go:126: ✓ Generated valid JSON: 2 nodes, 1 edges
=== RUN   TestE2E_FullWorkflow/4_GenerateCypherOutput
    e2e_test.go:173: ✓ Generated valid Cypher output (570 bytes)
=== RUN   TestE2E_FullWorkflow/5_InsertIntoNeo4j
    e2e_test.go:193: ✓ Data inserted into Neo4j
=== RUN   TestE2E_FullWorkflow/6_VerifyNodesInDatabase
    e2e_test.go:205: ✓ Found 2 nodes in Neo4j
=== RUN   TestE2E_FullWorkflow/7_VerifySpecificResources
    e2e_test.go:236: ✓ Verified resource null_resource.cluster
    e2e_test.go:236: ✓ Verified resource null_resource.app
=== RUN   TestE2E_FullWorkflow/8_VerifyRelationships
    e2e_test.go:248: ✓ Found 1 DEPENDS_ON relationships in Neo4j
    e2e_test.go:255: ✓ Verified dependency: null_resource.app → null_resource.cluster
=== RUN   TestE2E_FullWorkflow/9_TestIdempotency
    e2e_test.go:283: ✓ Idempotency verified: 2 nodes before and after second update
=== RUN   TestE2E_FullWorkflow/10_QueryGraph
    e2e_test.go:331: ✓ Found 1 dependencies:
    e2e_test.go:333:   null_resource.app (null_resource) → null_resource.cluster (null_resource)
--- PASS: TestE2E_FullWorkflow (1.93s)
```

## Individual Test Suites

### Configuration Tests

Tests the `init config` and `check database` commands:

```bash
go test -v -run TestE2E_ConfigCommands
```

### Database Check Tests

Tests the database connectivity verification:

```bash
go test -v -run TestE2E_CheckDatabase
```

### Full Workflow Tests

Tests the complete end-to-end workflow:

```bash
go test -v -run TestE2E_FullWorkflow
```

## Running All Tests

To run both unit tests and E2E tests:

```bash
make test-all
```

## Continuous Integration

For CI environments, you'll need to:

1. Start Neo4j before running tests
2. Create a `.terraform-graphx.yaml` with test credentials
3. Run the E2E tests

Example GitHub Actions workflow:

```yaml
- name: Start Neo4j
  run: |
    docker run -d \
      --name neo4j \
      -p 7474:7474 -p 7687:7687 \
      -e NEO4J_AUTH=neo4j/testpassword \
      neo4j:latest
    sleep 10

- name: Setup config
  run: |
    echo 'neo4j:
      uri: bolt://localhost:7687
      user: neo4j
      password: testpassword' > .terraform-graphx.yaml

- name: Run tests
  run: make test-all
```

## Troubleshooting

### "Cannot connect to Neo4j"

- Ensure Neo4j is running: `docker ps` or check your Neo4j service
- Verify the URI in `.terraform-graphx.yaml` is correct
- Test manually: `./terraform-graphx check database`

### "Config file not found"

Run `./terraform-graphx init config` and edit the generated file.

### Tests timeout

Increase the timeout: `go test -v -run TestE2E -timeout 5m`

### Database has old data

The E2E tests clear the database at the start, but you can manually clear it:

```cypher
// In Neo4j Browser
MATCH (n:Resource) DETACH DELETE n
```

## Test Data

The E2E tests use the Terraform configuration in the `examples/` directory:

- `null_resource.cluster` - A resource with triggers
- `null_resource.app` - A resource that depends on cluster

This creates a simple dependency graph to verify all functionality.
