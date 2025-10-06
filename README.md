# Terraform GraphX

A CLI tool that generates dependency graphs from your Terraform infrastructure and stores them in Neo4j for powerful querying and visualization.

![Example Graph](screenshoot.png)

## Quick Start

Get up and running in 3 simple steps:

```bash
# 1. Initialize configuration and Neo4j database
terraform-graphx init
terraform-graphx start

# 2. Generate and visualize your infrastructure graph
terraform-graphx --update

# 3. Open Neo4j Browser and explore
# Visit http://localhost:7474
# Username: neo4j
# Password: (shown during init)
```

## Features

- **🚀 Zero Configuration Start**: Built-in Docker support manages Neo4j automatically
- **📊 Multiple Output Formats**: Export as JSON, Cypher statements, or push directly to Neo4j
- **🔄 Idempotent Updates**: Run multiple times safely without duplicating data
- **🎯 Plan Support**: Analyze graphs from saved Terraform plans
- **🤖 AI-Ready**: Perfect foundation for AI agents via Model Context Protocol (MCP)
- **🔒 Secure by Default**: Auto-generated passwords, automatic `.gitignore` entries

## Installation

### Option 1: Download Pre-built Binary (Recommended)

Download the latest release for your platform from the [GitHub Releases](https://github.com/daniellvog/terraform-graphx/releases) page.

**Linux/macOS:**

```bash
# Download and extract (replace URL with latest release)
wget <URL_TO_TAR.GZ>
tar -xzf terraform-graphx_*.tar.gz

# Install to system path
sudo mv terraform-graphx /usr/local/bin/

# Verify installation
terraform-graphx --help
```

**Windows:**

1. Download the `.zip` file for Windows
2. Extract `terraform-graphx.exe`
3. Add the directory to your system's PATH

### Option 2: Build from Source

Requirements: Go 1.22 or later

```bash
git clone https://github.com/daniellvog/terraform-graphx.git
cd terraform-graphx
go build -o terraform-graphx .
sudo mv terraform-graphx /usr/local/bin/
```

## Basic Usage

### Prerequisites

- Terraform installed and project initialized (`terraform init`)
- Docker installed (for built-in Neo4j support)

### Generate JSON Output

Export your infrastructure graph to JSON:

```bash
terraform-graphx > graph.json
```

**Example output:**

```json
{
  "nodes": [
    {
      "id": "null_resource.cluster",
      "type": "null_resource",
      "provider": "registry.terraform.io/hashicorp/null",
      "name": "cluster"
    },
    {
      "id": "null_resource.app",
      "type": "null_resource",
      "provider": "registry.terraform.io/hashicorp/null",
      "name": "app"
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

### Generate Cypher Statements

Export as Neo4j-compatible Cypher statements:

```bash
terraform-graphx --format=cypher > graph.cypher
```

**Example output:**

```cypher
MERGE (n:Resource {id: 'null_resource.cluster'})
SET n.type = 'null_resource', n.provider = 'registry.terraform.io/hashicorp/null', n.name = 'cluster';

MERGE (n:Resource {id: 'null_resource.app'})
SET n.type = 'null_resource', n.provider = 'registry.terraform.io/hashicorp/null', n.name = 'app';

MATCH (from:Resource {id: 'null_resource.app'}), (to:Resource {id: 'null_resource.cluster'})
MERGE (from)-[:DEPENDS_ON]->(to);
```

### Update Neo4j Directly

Push your infrastructure graph to Neo4j:

```bash
# Using credentials from .terraform-graphx.yaml
terraform-graphx --update

# Or specify credentials manually
terraform-graphx --update \
  --neo4j-uri="bolt://localhost:7687" \
  --neo4j-user="neo4j" \
  --neo4j-pass="your-password"
```

### Analyze Terraform Plans

Generate graphs from saved plans without applying them:

```bash
terraform plan -out=tfplan
terraform-graphx --plan=tfplan --format=json
```

## CLI Reference

### Commands

#### `terraform-graphx` (default)

Generate and output infrastructure graph.

**Flags:**

- `--format <json|cypher>` - Output format (default: `json`)
- `--plan <file>` - Use existing Terraform plan file
- `--update` - Push graph to Neo4j database
- `--neo4j-uri <uri>` - Neo4j connection URI (default: `bolt://localhost:7687`)
- `--neo4j-user <user>` - Neo4j username (default: `neo4j`)
- `--neo4j-pass <password>` - Neo4j password

**Examples:**

```bash
# Output JSON to stdout
terraform-graphx

# Output Cypher statements
terraform-graphx --format=cypher

# Analyze a saved plan
terraform-graphx --plan=tfplan.binary

# Update Neo4j with custom credentials
terraform-graphx --update --neo4j-pass=secret
```

#### `terraform-graphx init`

Initialize configuration and data directory for your project.

**What it does:**

- Creates `.terraform-graphx.yaml` with secure random password
- Creates `neo4j-data/` directory for database persistence
- Adds both to `.gitignore` (if in a Git repository)

**Example:**

```bash
terraform-graphx init
```

#### `terraform-graphx start`

Start Neo4j Docker container with project-specific database.

**What it does:**

- Pulls Neo4j image (if not present)
- Starts container named `terraform-graphx-neo4j`
- Mounts local `neo4j-data/` directory
- Uses credentials from `.terraform-graphx.yaml`

**Example:**

```bash
terraform-graphx start
```

**Important:** If `neo4j-data/` contains existing data, Neo4j will use the password stored in that data, **not** the password in your config file.

#### `terraform-graphx stop`

Stop and remove Neo4j container (preserves data).

**Example:**

```bash
terraform-graphx stop
```

#### `terraform-graphx check database`

Verify Neo4j connection and credentials.

**Example:**

```bash
terraform-graphx check database
```

## Configuration

### Configuration File

`terraform-graphx init` creates a `.terraform-graphx.yaml` file:

```yaml
neo4j:
  uri: bolt://localhost:7687
  user: neo4j
  password: <randomly-generated-password>
  docker_image: neo4j:community
```

**⚠️ Security Note:** This file contains sensitive credentials and is automatically added to `.gitignore`.

### Configuration Priority

Settings are loaded in this order (highest to lowest priority):

1. **Command-line flags** - Override everything
2. **Configuration file** - `.terraform-graphx.yaml`
3. **Default values** - Built-in defaults

### Customizing Neo4j Image

Edit `.terraform-graphx.yaml` to use a specific Neo4j version:

```yaml
neo4j:
  docker_image: neo4j:5.15.0  # Use specific version
```

## Neo4j Database Management

### Project-Specific Databases

Neo4j Community Edition supports only one database per instance. `terraform-graphx` solves this by using project-specific data volumes (`neo4j-data/` in each project directory).

**Benefits:**

- Isolated graphs per Terraform project
- Independent start/stop for each project
- No conflicts between infrastructure states

### Working with Multiple Projects

```bash
# Project A
cd ~/projects/infrastructure-a
terraform-graphx init
terraform-graphx start
terraform-graphx --update

# Switch to Project B
cd ~/projects/infrastructure-b
terraform-graphx init
terraform-graphx start  # Automatically uses Project B's data
terraform-graphx --update

# Return to Project A
cd ~/projects/infrastructure-a
terraform-graphx start  # Reconnects to Project A's data
```

### Handling Existing Data

If you encounter authentication errors with existing data:

#### Option 1: Fresh Start (Development/Testing)

```bash
terraform-graphx stop
sudo rm -rf neo4j-data
rm -f .terraform-graphx.yaml
terraform-graphx init
terraform-graphx start
```

#### Option 2: Use Existing Password (Production)

Edit `.terraform-graphx.yaml` and update the password to match your existing database.

### Manual Docker Setup (Advanced)

If you prefer manual Docker management:

```bash
# Create data directory
mkdir neo4j-data

# Run Neo4j container
docker run -d \
  --name terraform-graphx-neo4j \
  -p 7474:7474 -p 7687:7687 \
  -v $(pwd)/neo4j-data:/data \
  -e NEO4J_AUTH=neo4j/your-password \
  neo4j:community

# Update .terraform-graphx.yaml with your password
# Then use: terraform-graphx --update
```

## Advanced Use Cases

### AI-Enhanced Infrastructure Management

`terraform-graphx` enables AI agents to understand and interact with your infrastructure through the **Model Context Protocol (MCP)**.

**Architecture:**

```text
Terraform Infrastructure → terraform-graphx → Neo4j Graph Database
                                                      ↓
                                              Model Context Protocol
                                                      ↓
                                              AI Agents (Gemini, etc.)
```

**Capabilities enabled:**

- **Context Understanding**: AI agents query the graph to understand relationships between hundreds/thousands of resources
- **Impact Analysis**: Predict change ripple effects before applying Terraform plans
- **Autonomous Operations**: AI generates plans and executes changes with full infrastructure context
- **Knowledge Base**: Infrastructure becomes queryable knowledge rather than static files

**Example MCP Queries:**

```cypher
// PROMPT: What will break if I delete this resource?
MATCH (dependent:Resource)-[:DEPENDS_ON*]->(target:Resource {id: 'aws_vpc.main'})
RETURN dependent.id

// PROMPT: Find all resources of a specific type
MATCH (n:Resource {type: 'aws_instance'})
RETURN n.id, n.name

// PROMPT: Detect circular dependencies
MATCH path = (a:Resource)-[:DEPENDS_ON*]->(a)
RETURN path
```

## Development

### Project Structure

```text
cmd/                    # Cobra CLI command definitions
  ├── root.go          # Main entrypoint
  ├── graphx.go        # Graph generation command
  ├── init.go          # Configuration initialization
  ├── start.go         # Neo4j container start
  ├── stop.go          # Neo4j container stop
  └── check.go         # Database connectivity check

internal/
  ├── runner/          # Orchestrates terraform graph workflow
  ├── config/          # Configuration loading and merging
  ├── parser/          # DOT to JSON graph parsing
  ├── formatter/       # JSON and Cypher output formatters
  ├── neo4j/           # Neo4j client and database operations
  └── graph/           # Graph data structures
```

### Building

```bash
# Build binary
make build

# Run unit tests
make test-unit

# Run end-to-end tests (requires Neo4j)
make test-e2e

# Run all tests
make test-all

# Clean build artifacts
make clean
```

### Testing

**Unit Tests:**

```bash
go test -v -short ./...
```

**E2E Tests:**

Require running Neo4j instance with credentials in `.terraform-graphx.yaml`:

```bash
# Setup
terraform-graphx init
terraform-graphx start

# Run tests
make test-e2e
```

### Adding New Output Formats

1. Add formatter function in `internal/formatter/formatter.go`
2. Wire into `runner.formatAndPrintGraph` in `internal/runner/runner.go`
3. Add flag option in `cmd/graphx.go`

Example:

```go
// internal/formatter/formatter.go
func ToYAML(graph *graph.Graph) (string, error) {
    // Implementation
}

// internal/runner/runner.go
case "yaml":
    output, err = formatter.ToYAML(graphData)

// cmd/graphx.go
cmd.Flags().String("format", "json", "Output format (json, cypher, yaml)")
```

## License

[Include your license information here]

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## Support

- **Issues**: [GitHub Issues](https://github.com/daniellvog/terraform-graphx/issues)
- **Discussions**: [GitHub Discussions](https://github.com/daniellvog/terraform-graphx/discussions)
