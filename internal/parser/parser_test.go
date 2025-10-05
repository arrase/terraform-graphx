package parser

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseFromData(t *testing.T) {
	// Read the sample JSON data from the testdata file
	path := filepath.Join("testdata", "sample_plan.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read test data file: %v", err)
	}

	// Parse the data
	plan, err := ParseFromData(data)
	if err != nil {
		t.Fatalf("ParseFromData failed: %v", err)
	}

	// Assertions to verify the plan was parsed correctly
	if plan == nil {
		t.Fatal("Parsed plan should not be nil")
	}

	// Check for the correct number of resources in planned_values
	if len(plan.PlannedValues.RootModule.Resources) != 2 {
		t.Errorf("Expected 2 resources in planned_values, got %d", len(plan.PlannedValues.RootModule.Resources))
	}

	// Check for the correct number of resources in configuration
	if len(plan.Configuration.RootModule.Resources) != 2 {
		t.Errorf("Expected 2 resources in configuration, got %d", len(plan.Configuration.RootModule.Resources))
	}

	// Check a specific resource's properties
	var appResource Resource
	for _, r := range plan.PlannedValues.RootModule.Resources {
		if r.Address == "null_resource.app" {
			appResource = r
			break
		}
	}
	if appResource.Address == "" {
		t.Fatal("Could not find resource 'null_resource.app' in planned_values")
	}
	if appResource.Type != "null_resource" {
		t.Errorf("Expected resource type 'null_resource', got '%s'", appResource.Type)
	}
	if appResource.Name != "app" {
		t.Errorf("Expected resource name 'app', got '%s'", appResource.Name)
	}
}