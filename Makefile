.PHONY: help build test test-unit test-e2e test-all clean install

# Default target
help:
	@echo "Available targets:"
	@echo "  build       - Build the terraform-graphx binary"
	@echo "  test        - Run unit tests only"
	@echo "  test-unit   - Run unit tests only (same as test)"
	@echo "  test-e2e    - Run end-to-end tests (requires Neo4j)"
	@echo "  test-all    - Run all tests (unit + e2e)"
	@echo "  clean       - Remove built binaries"
	@echo "  install     - Install terraform-graphx to /usr/local/bin"
	@echo ""
	@echo "E2E tests require:"
	@echo "  - Neo4j running with credentials in .terraform-graphx.yaml"
	@echo "  - Run 'terraform graphx init config' to create the config file"
	@echo "  - Edit .terraform-graphx.yaml with your Neo4j credentials"

# Build the binary
build:
	@echo "Building terraform-graphx..."
	go build -o terraform-graphx .
	@echo "✓ Build complete: ./terraform-graphx"

# Run unit tests only
test: test-unit

test-unit:
	@echo "Running unit tests..."
	go test -v -short ./...

# Run end-to-end tests
test-e2e: build
	@echo "Running end-to-end tests..."
	@echo "Note: This requires Neo4j to be running with credentials in .terraform-graphx.yaml"
	@if [ ! -f .terraform-graphx.yaml ]; then \
		echo "ERROR: .terraform-graphx.yaml not found!"; \
		echo "Run: ./terraform-graphx init config"; \
		echo "Then edit .terraform-graphx.yaml with your Neo4j credentials"; \
		exit 1; \
	fi
	@if [ ! -f examples/.terraform-graphx.yaml ]; then \
		echo "Copying config to examples directory..."; \
		cp .terraform-graphx.yaml examples/.terraform-graphx.yaml; \
	fi
	go test -v -run TestE2E -timeout 3m ./...

# Run all tests
test-all: build
	@echo "Running all tests (unit + e2e)..."
	@if [ ! -f examples/.terraform-graphx.yaml ] && [ -f .terraform-graphx.yaml ]; then \
		cp .terraform-graphx.yaml examples/.terraform-graphx.yaml; \
	fi
	go test -v -timeout 3m ./...

# Clean built files
clean:
	@echo "Cleaning..."
	rm -f terraform-graphx
	rm -f examples/.terraform-graphx.yaml
	@echo "✓ Clean complete"

# Install to system
install: build
	@echo "Installing terraform-graphx to /usr/local/bin..."
	sudo mv terraform-graphx /usr/local/bin/
	@echo "✓ Installation complete"
	@echo "You can now use: terraform graphx"
