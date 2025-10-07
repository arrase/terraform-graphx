# Copilot Instructions for terraform-graphx

## Architecture Overview
This CLI converts Terraform dependency graphs into Neo4j graph databases. The pipeline is:
1. Shell out to `terraform graph` to get DOT format
2. Parse DOT with `gographviz` → convert to JSON mimicking `dot -Tjson` output
3. Parse JSON into `graph.Graph` structs (nodes + edges)
4. Push to Neo4j via parameterized queries using `ToCypherTransaction`

**Critical**: Neo4j writes MUST use `ToCypherTransaction` (UNWIND + params)—prevents injection and improves performance.

## Data Flow & Key Components

### CLI Entry Point (`cmd/`)
- `cmd/root.go`: Displays help only; subcommands registered via `init()` in their files
- `cmd/update.go`: Sets `cfg.Update = true`, then runs `runner.Run` → pushes to Neo4j
- `cmd/init.go`: Generates `.terraform-graphx.yaml` + random 16-char password, creates `neo4j-data/`, updates `.gitignore`
- `cmd/start.go`: Uses Docker Go SDK (not shell) to pull `neo4j:community`, mount `neo4j-data` → `/data`, warns about existing data/password conflicts
- `cmd/stop.go`: Removes container but keeps volume
- `cmd/check.go`: Validates Neo4j connectivity via `VerifyConnectivity`

### Graph Pipeline (`internal/runner/runner.go`)
```go
generateGraphData(planFile) → terraform graph [-plan=file] 
  → gographviz.Parse(DOT) 
  → convertGraphToJSON() // mimics dot -Tjson
  → parser.ParseGraph(JSON bytes)
  → handleOutput() // updates Neo4j database
```

**Node ID extraction**: `parser.cleanLabel` strips `["..."]` quoting from DOT labels; expects Terraform resource addresses like `aws_instance.web`.

### Graph Structure (`internal/graph/graph.go`)
- Nodes: `id` (resource address), `type` (e.g., `aws_instance`), `provider`, `name`, optional `attributes`
- Edges: Always `DEPENDS_ON` relation between `from` → `to`

### Formatters (`internal/formatter/formatter.go`)
- `ToCypherTransaction`: **Use for Neo4j driver execution** → UNWIND with `$nodes` and `$edges` params

### Neo4j Client (`internal/neo4j/client.go`)
`UpdateGraph` pattern (idempotent):
1. `fetchExistingResourceIDs`: `MATCH (n:Resource) RETURN n.id`
2. `deleteObsoleteResources`: `DETACH DELETE` anything not in new graph
3. `upsertGraph`: Execute parameterized transaction from `ToCypherTransaction`

**Schema**: All nodes have `Resource` label + `id/type/name/provider` properties (no indexes defined yet).

## Configuration System (`internal/config/config.go`)
Precedence: **CLI flags > `.terraform-graphx.yaml` > defaults**
- `Load()`: Searches cwd → `$HOME` for `.terraform-graphx.yaml`
- `LoadAndMerge(cmd, args)`: Overlays flags onto loaded config
- `Save()`: Enforces 0600 permissions for security (password stored in plaintext)

**Default values**: `bolt://localhost:7687`, user `neo4j`, no update mode

## Developer Workflows

### Build & Test
```bash
make build              # → ./terraform-graphx binary
make test-unit          # go test -v -short ./... (no Neo4j needed)
make test-e2e           # requires ./terraform-graphx binary + .terraform-graphx.yaml + Terraform CLI
```

**E2E test flow**:
1. Checks for binary, copies config to `examples/`
2. Runs `terraform init` in `examples/` if needed
3. Tests Neo4j update with live connection (skips if password empty or connectivity fails)
4. Verifies idempotency and graph structure

### Adding a New CLI Command
1. Create `cmd/newcmd.go` with `cobra.Command`
2. Register in `init()`: `rootCmd.AddCommand(newCmd)`
3. Use `config.LoadAndMerge` if config needed
4. Return errors via `RunE` (Cobra handles exit codes)

## Project-Specific Patterns

### Docker Management (No Shell Commands)
- Always use `github.com/docker/docker/client` SDK
- Container name: `terraform-graphx-neo4j`
- Volume mount: `$(pwd)/neo4j-data` → `/data` (container path)
- `start.go` checks for existing data in `neo4j-data/dbms/` and warns about password mismatch

### Error Handling
- Wrap with context: `fmt.Errorf("failed to parse graph: %w", err)`
- Cobra `RunE` surfaces errors automatically
- Progress logging: `log.Println("Generating Terraform graph...")`

### Testing Fixtures
- `examples/` contains `main.tf` for end-to-end validation
- E2E tests verify Neo4j state and graph structure
- Unit tests use `-short` flag to skip integration tests

## Common Gotchas
- **Plan file support**: Pass `--plan=tfplan.binary` to analyze saved plans instead of current state
- **Neo4j Community limitation**: One database per instance → use project-specific `neo4j-data/` volumes
- **Password security**: `.terraform-graphx.yaml` auto-added to `.gitignore` during `init`, but manual setups may expose credentials
- **Graph parsing**: Labels must match Terraform resource address format; `cleanLabel` regex expects `["resource.name"]` pattern
- **Update-only mode**: The CLI only supports pushing to Neo4j via the `update` command. There is no standalone JSON/Cypher output mode.
