# Terraform GraphX

`terraform-graphx` is a custom Terraform CLI extension that generates a dependency graph of your Terraform resources and can update a Neo4j database with the infrastructure state. This tool allows you to visualize and query your infrastructure as a graph without modifying Terraform's core.

## Features

-   **Standalone Binary**: A single Go binary that acts as a Terraform subcommand (`terraform graphx`).
-   **Machine-Readable Output**: Generates a dependency graph in JSON or Cypher format.
-   **Neo4j Integration**: Can directly update a Neo4j database with the graph data.
-   **Modular Design**: Built with separate components for parsing, graph building, and output formatting.

## Installation

You can install `terraform-graphx` by downloading a pre-compiled binary or by building it from source.

### Pre-compiled Binaries (Recommended)

The easiest way to install `terraform-graphx` is to download the latest release from the **GitHub Releases** page for this repository.

1.  Download the archive for your operating system and architecture.
2.  Extract the `terraform-graphx` binary.
3.  Move the binary to a directory in your system's `PATH`. The binary must be named `terraform-graphx` for Terraform to detect it as a subcommand.

**Example for Linux/macOS:**
```bash
# Replace with the correct URL from the releases page
wget <URL_TO_TAR.GZ>
tar -xzf terraform-graphx_*.tar.gz
sudo mv terraform-graphx /usr/local/bin/
```

### Build from Source

If you have Go (version 1.22+) installed, you can build `terraform-graphx` from source.

1.  Clone the repository:
    ```bash
    git clone https://github.com/daniellvog/terraform-graphx.git
    cd terraform-graphx
    ```

2.  Build the binary:
    ```bash
    go build -o terraform-graphx .
    ```

3.  Place the binary in your `PATH`:
    ```bash
    sudo mv terraform-graphx /usr/local/bin/
    ```

4.  Verify the installation:
    ```bash
    terraform -help
    ```

## Usage

Navigate to your Terraform project directory and run the `graphx` subcommand.

### Prerequisites

You must have `terraform` installed and have initialized your project with `terraform init`.

### Generating a JSON Graph

By default, `terraform-graphx` outputs the dependency graph in JSON format.

```bash
terraform graphx > graph.json
```

**Example JSON Output:**
```json
{
  "nodes": [
    {
      "id": "null_resource.cluster",
      "type": "null_resource",
      "provider": "registry.terraform.io/hashicorp/null",
      "name": "cluster",
      "attributes": {
        "id": "...",
        "triggers": {
          "cluster_name": "my-cluster"
        }
      }
    },
    {
      "id": "null_resource.app",
      "type": "null_resource",
      "provider": "registry.terraform.io/hashicorp/null",
      "name": "app",
      "attributes": {
        "id": "...",
        "triggers": {
          "cluster_id": "..."
        }
      }
    }
  ],
  "edges": [
    {
      "from": "null_resource.app",
      "to": "null_resource.cluster",
      "relation": "DEPENDS_ON"
    }
  ]
}
```

### Generating Cypher Statements

You can also output the graph as a series of Cypher `MERGE` statements, which are idempotent.

```bash
terraform graphx --format=cypher > graph.cypher
```

**Example Cypher Output:**
```cypher
MERGE (n:Resource {id: 'null_resource.cluster'})
SET n.type = 'null_resource', n.provider = 'registry.terraform.io/hashicorp/null', n.name = 'cluster';
MERGE (n:Resource {id: 'null_resource.app'})
SET n.type = 'null_resource', n.provider = 'registry.terraform.io/hashicorp/null', n.name = 'app';

MATCH (from:Resource {id: 'null_resource.app'}), (to:Resource {id: 'null_resource.cluster'})
MERGE (from)-[:DEPENDS_ON]->(to);
```

### Updating a Neo4j Database

The `--update` flag allows you to push the graph directly into a Neo4j database.

```bash
export NEO4J_PASS="your-secret-password"
terraform graphx --update \
  --neo4j-uri="bolt://localhost:7687" \
  --neo4j-user="neo4j" \
  --neo4j-pass="$NEO4J_PASS"
```

The tool uses idempotent `MERGE` statements, so you can run this command multiple times without creating duplicate nodes or relationships.

## CLI Flags

-   `--format <format>`: The output format for the graph. Can be `json` (default) or `cypher`.
-   `--plan <file>`: Path to a pre-generated Terraform plan file. If not provided, `terraform-graphx` will generate one.
-   `--update`: A boolean flag to enable updating a Neo4j database.
-   `--neo4j-uri <uri>`: The URI for the Neo4j database (e.g., `bolt://localhost:7687`).
-   `--neo4j-user <user>`: The username for the Neo4j database.
-   `--neo4j-pass <password>`: The password for the Neo4j database. Can also be set via an environment variable.

## Development

This project is written in Go and uses the Cobra library for the CLI.

-   **Parser**: `internal/parser/parser.go` - Executes `terraform show -json` and parses the output.
-   **Builder**: `internal/builder/builder.go` - Constructs the graph from the parsed data.
-   **Formatter**: `internal/formatter/` - Contains logic for JSON and Cypher output.
-   **Neo4j Client**: `internal/neo4j/client.go` - Handles communication with the Neo4j database.
-   **Main Command**: `cmd/graphx.go` - Ties everything together.