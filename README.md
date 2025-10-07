# Terraform GraphX

A CLI tool that generates dependency graphs from your Terraform infrastructure and stores them in Neo4j for powerful querying and visualization.

![Example Graph](screenshoot.png)

## Features

- **🚀 Zero Configuration Start**: Built-in Docker support manages Neo4j automatically
- **📊 Multiple Output Formats**: Export as JSON, Cypher statements, or push directly to Neo4j
- **🔄 Idempotent Updates**: Run multiple times safely without duplicating data
- **🎯 Plan Support**: Analyze graphs from saved Terraform plans
- **🤖 AI-Ready**: Perfect foundation for AI agents via Model Context Protocol (MCP)
- **🔒 Secure by Default**: Auto-generated passwords, automatic `.gitignore` entries

## AI-Enhanced Infrastructure Management

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

### Quick Start

Get up and running in 3 simple steps:

```bash
# 1. Initialize configuration and Neo4j database
terraform-graphx init
terraform-graphx start

# 2. Generate and visualize your infrastructure graph
terraform-graphx update

# 3. Open Neo4j Browser and explore
# Visit http://localhost:7474
# Username: neo4j
# Password: (shown during init)
```

## Configuration File

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
terraform-graphx update

# Switch to Project B
cd ~/projects/infrastructure-b
terraform-graphx init
terraform-graphx start  # Automatically uses Project B's data
terraform-graphx update

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
# Then use: terraform-graphx update
```

## Development

### Project Structure

```text
cmd/                    # Cobra CLI command definitions
  ├── root.go          # Main entrypoint and graph generation
  ├── update.go        # Neo4j update command
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

## License

[Include your license information here]

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## Support

- **Issues**: [GitHub Issues](https://github.com/daniellvog/terraform-graphx/issues)
- **Discussions**: [GitHub Discussions](https://github.com/daniellvog/terraform-graphx/discussions)
