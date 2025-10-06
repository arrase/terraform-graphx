# Copilot Instructions for terraform-graphx

## Project Overview
terraform-graphx is a Go CLI tool that parses Terraform infrastructure graphs and exports them to Neo4j databases. It's designed to enable AI agents (via Model Context Protocol) to query and understand infrastructure dependencies at scale.

## Architecture

### Data Flow Pipeline
1. **Input**: `terraform graph` → DOT format → go-graphviz library → JSON graph data
2. **Parse**: `internal/parser` converts JSON to internal `graph.Graph` structure
3. **Output**: `internal/formatter` converts to JSON/Cypher OR `internal/neo4j` updates database

### Key Components
- **cmd/**: Cobra CLI commands (`root.go`, `graphx.go`, `init.go`, `start.go`, `stop.go`, `check.go`)
- **internal/runner**: Orchestrates `terraform graph` + `dot` pipeline, delegates to formatter/neo4j
- **internal/parser**: Parses JSON output from go-graphviz; extracts nodes/edges from DOT graph JSON
- **internal/formatter**: 
  - `ToJSON()`: Pretty-print graph
  - `ToCypher()`: Generate string-based MERGE statements (for files)
  - `ToCypherTransaction()`: Parameterized queries with `UNWIND` batching (for driver - prevents injection)
- **internal/neo4j**: Manages Neo4j driver; implements idempotent updates (deletes obsolete resources, upserts current)
- **internal/config**: Viper-based config with priority: CLI flags > `.terraform-graphx.yaml` > defaults

### Neo4j Update Strategy
The `UpdateGraph()` method (internal/neo4j/client.go):
1. Fetches existing resource IDs from Neo4j
2. Computes diff: deletes resources not in new graph (with `DETACH DELETE`)
3. Uses parameterized `UNWIND` batching to upsert nodes and relationships
4. **Critical**: Always uses `ToCypherTransaction()` for driver execution (prevents injection), never `ToCypher()`

## Development Workflows

### Building & Testing
```bash
make build              # Builds ./terraform-graphx binary
make test-unit          # Unit tests only (no Neo4j): go test -v -short ./...
make test-e2e           # E2E tests (requires Neo4j + .terraform-graphx.yaml)
make test-all           # Both unit + E2E
```

### E2E Test Setup
1. Run `./terraform-graphx init` to create `.terraform-graphx.yaml` and `neo4j-data/`
2. Run `./terraform-graphx start` to launch Docker Neo4j container
3. Run `make test-e2e` (copies config to `examples/` if needed)
4. Tests use `examples/main.tf` with `terraform init` in that directory

**Important**: E2E tests skip if `.terraform-graphx.yaml` missing or password empty. Check `e2e_test.go:TestE2E_FullWorkflow` for test structure.

### Docker Neo4j Management
- `terraform-graphx init`: Creates config + `neo4j-data/` directory + adds to `.gitignore`
- `terraform-graphx start`: Pulls `neo4j:community` image, starts container with volume mount to `./neo4j-data`
- `terraform-graphx stop`: Stops and removes container (data persists in `neo4j-data/`)
- **Password Caveat**: Neo4j ignores config password if `neo4j-data/dbms/` exists (uses existing DB password)

### Configuration Pattern
```go
// Load config in commands via:
cfg, err := config.LoadAndMerge(cmd, args)  // Merges file + CLI flags
```

Priority: CLI flags override `.terraform-graphx.yaml` override `DefaultConfig()`.

## Code Conventions

### Error Handling
- Wrap errors with context: `fmt.Errorf("failed to X: %w", err)`
- Runner functions return errors; main command `RunE` handles them
- Neo4j operations return detailed errors for connectivity/query failures

### Testing Patterns
- Use `testing.Short()` to skip E2E tests: `if testing.Short() { t.Skip(...) }`
- E2E tests verify Neo4j first, clear DB, run commands, validate results
- Config loading: E2E tests call `config.Load()` directly (no CLI flags)

### Graph Structure
```go
// internal/graph/graph.go
type Node struct {
    ID       string                 // e.g., "aws_vpc.main"
    Type     string                 // e.g., "aws_vpc"
    Provider string                 // e.g., "aws" (not in graph, empty)
    Name     string                 // e.g., "main"
}
type Edge struct {
    From, To string  // Node IDs
    Relation string  // Always "DEPENDS_ON"
}
```

### Neo4j Cypher Patterns
- **String escaping**: `ToCypher()` uses `escapeString()` to escape single quotes (`'` → `\'`)
- **Parameterized queries**: Always use `ToCypherTransaction()` for driver execution (returns query + params map)
- **Batch operations**: Use `UNWIND $nodes AS node_data` for efficient bulk upserts

## Common Tasks

### Adding New Output Format
1. Add formatter function in `internal/formatter/formatter.go`
2. Add case in `runner.formatAndPrintGraph()` switch statement
3. Update `--format` flag help text in `cmd/graphx.go`

### Adding New CLI Command
1. Create new file in `cmd/` (e.g., `cmd/newcmd.go`)
2. Define `cobra.Command` with `Use`, `Short`, `Long`, `RunE`
3. Register in `init()`: `rootCmd.AddCommand(newCmd)`
4. Follow config pattern: call `config.LoadAndMerge()` if needs config

### External Dependencies
- **Required external tools**: `terraform` (for graph generation)
- **Docker SDK**: Uses `github.com/docker/docker` for managing Neo4j containers (see `cmd/start.go`)
- **Neo4j Driver**: `github.com/neo4j/neo4j-go-driver/v5` - use `ExecuteWrite()` for transactions

## AI Agent Integration Context
This tool's primary purpose is to populate Neo4j for Model Context Protocol (MCP) queries. When modifying graph structure or Neo4j schema:
- Keep `Resource` node label consistent (all nodes are `:Resource`)
- Preserve `id`, `type`, `name`, `provider` properties for MCP queries
- Maintain `DEPENDS_ON` relationship type for dependency traversal
