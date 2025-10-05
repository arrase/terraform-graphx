package parser

import (
	"encoding/json"
	"fmt"
	"os/exec"
)

const (
	terraformCmd    = "terraform"
	showSubCmd      = "show"
	planSubCmd      = "plan"
	jsonFlag        = "-json"
	outFlag         = "-out"
	defaultPlanFile = "tfplan.binary"
)

// TerraformPlan represents the structure of the JSON output from `terraform show -json`.
type TerraformPlan struct {
	PlannedValues   PlannedValues    `json:"planned_values"`
	Configuration   Configuration    `json:"configuration"`
	ResourceChanges []ResourceChange `json:"resource_changes"`
}

// PlannedValues represents the planned state of resources.
type PlannedValues struct {
	RootModule Module `json:"root_module"`
}

// Module represents a Terraform module, which can contain resources and child modules.
type Module struct {
	Resources    []Resource `json:"resources"`
	ChildModules []Module   `json:"child_modules"`
}

// Resource represents a single resource in the plan.
type Resource struct {
	Address      string                 `json:"address"`
	Mode         string                 `json:"mode"`
	Type         string                 `json:"type"`
	Name         string                 `json:"name"`
	ProviderName string                 `json:"provider_name"`
	Values       map[string]interface{} `json:"values"`
}

// Configuration represents the parsed Terraform configuration.
type Configuration struct {
	RootModule ConfigModule `json:"root_module"`
}

// ConfigModule represents a module within the configuration.
type ConfigModule struct {
	Resources   []ConfigResource      `json:"resources"`
	ModuleCalls map[string]ModuleCall `json:"module_calls"`
}

// ConfigResource represents a resource block in the configuration.
type ConfigResource struct {
	Address     string                `json:"address"`
	Expressions map[string]Expression `json:"expressions"`
}

// ModuleCall represents a module block in the configuration.
type ModuleCall struct {
	Expressions map[string]Expression `json:"expressions"`
	Module      ConfigModule          `json:"module"`
}

// Expression represents a value or reference in the configuration.
type Expression struct {
	References []string `json:"references"`
}

// ResourceChange represents a planned change for a resource.
type ResourceChange struct {
	Address      string `json:"address"`
	Change       Change `json:"change"`
	ActionReason string `json:"action_reason"`
}

// Change represents the details of a resource change.
type Change struct {
	Actions []string `json:"actions"`
}

// Parse executes `terraform show -json` and unmarshals the output.
// If planFile is empty, it generates a new plan first.
func Parse(planFile string) (*TerraformPlan, error) {
	var cmd *exec.Cmd

	if planFile != "" {
		cmd = exec.Command(terraformCmd, showSubCmd, jsonFlag, planFile)
	} else {
		// Generate a plan if not provided (requires `terraform init`)
		if err := generatePlan(); err != nil {
			return nil, fmt.Errorf("failed to generate terraform plan: %w", err)
		}
		cmd = exec.Command(terraformCmd, showSubCmd, jsonFlag, defaultPlanFile)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("terraform command failed: %w\nOutput: %s", err, string(output))
	}

	return ParseFromData(output)
}

// generatePlan creates a new Terraform plan file.
func generatePlan() error {
	cmd := exec.Command(terraformCmd, planSubCmd, outFlag+"="+defaultPlanFile)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%w\nOutput: %s", err, string(output))
	}
	return nil
}

// ParseFromData unmarshals a Terraform plan from a byte slice.
// This is exported for testing purposes.
func ParseFromData(data []byte) (*TerraformPlan, error) {
	var plan TerraformPlan
	if err := json.Unmarshal(data, &plan); err != nil {
		return nil, fmt.Errorf("failed to unmarshal terraform plan JSON: %w", err)
	}
	return &plan, nil
}
