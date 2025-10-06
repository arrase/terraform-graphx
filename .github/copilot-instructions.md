# Copilot Instructions for terraform-graphx

## Essentials
- CLI entrypoint lives in `cmd/root.go`; the root command only displays help and available subcommands.
- The `update` subcommand in `cmd/update.go` forces `cfg.Update = true`, loads config via `config.LoadAndMerge`, and executes the graph update flow.
- Data flow: `runner.generateGraphData` shells out to `terraform graph` (optionally `-plan=<file>`), feeds DOT into `gographviz`, and `convertGraphToJSON` mimics `dot -Tjson` for the parser.
- Parsed graphs use `internal/graph/graph.go`; nodes expose `id/type/provider/name` (attributes optional) and edges are always `DEPENDS_ON`.
- `internal/parser.ParseGraph` strips DOT quoting with `cleanLabel`, so keep labels consistent with Terraform resource addresses.

## Key packages & patterns
- `internal/runner.handleOutput` either prints via `formatter` or calls `updateNeo4jDatabase`; the `update` subcommand demands populated `neo4j` config and passes through `validateNeo4jConfig`.
- Formatter options: `ToJSON` for pretty JSON, `ToCypher` for file-friendly MERGEs, and `ToCypherTransaction` (UNWIND + params) for anything executed through the driver—never send raw `ToCypher` strings to Neo4j.
- `internal/neo4j.Client.UpdateGraph` fetches current `Resource` ids, `DETACH DELETE`s anything missing, then executes the parameterized upsert. Always preserve the `Resource` label plus `id/type/name/provider` properties.
- Configuration precedence is flags > `.terraform-graphx.yaml` > defaults; `config.Load` searches cwd then `$HOME`. Saving the config enforces 0600 permissions.

## CLI workflows
- `terraform-graphx init` generates config + random password, creates `neo4j-data/`, and appends both paths to `.gitignore` when inside a git repo.
- `terraform-graphx start` uses the Docker SDK (no shelling out) to pull `neo4j:community`, mount `neo4j-data` to `/data`, and warns if existing data will override the stored password.
- `terraform-graphx stop` removes the managed container but keeps the volume; `check database` reuses `neo4j.NewClient` + `VerifyConnectivity` to confirm credentials.
- Typical graph usage: run in a Terraform project (after `terraform init`); use `terraform-graphx update` to push the graph directly to Neo4j (supports `--plan` flag for plan files).

## Builds, tests, and fixtures
- `make build` emits the `./terraform-graphx` binary; unit tests run with `make test-unit` (`go test -v -short ./...`).
- `make test-e2e` first ensures the binary exists and copies `.terraform-graphx.yaml` into `examples/`. The suite in `e2e_test.go` requires Neo4j credentials plus Terraform CLI and will skip when the password is empty or connectivity fails.
- Examples live in `examples/` and are used for JSON/Cypher fixture expectations as well as end-to-end runs.

## Coding conventions & extension tips
- Errors are wrapped with context (`fmt.Errorf("failed to …: %w", err)`) and surfaced by Cobra `RunE` handlers; logging uses `log.Println` for progress messages.
- When extending graph output (new format), add a formatter function, wire it into `runner.formatAndPrintGraph`.
- New CLI commands follow the existing pattern: create a file in `cmd/`, define a `cobra.Command`, register it in `init()`, and call `config.LoadAndMerge` if configuration is needed.
- Keep Docker interactions in Go (see `cmd/start.go`/`stop.go`) and prefer the existing client abstractions for Neo4j updates instead of crafting ad-hoc Cypher strings.
