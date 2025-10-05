package parser

import (
	"encoding/json"
	"os/exec"
)

// TerraformPlan represents the structure of the JSON output from `terraform show -json`.
type TerraformPlan struct {
	PlannedValues   PlannedValues   `json:"planned_values"`
	Configuration   Configuration   `json:"configuration"`
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
	Resources    []ConfigResource `json:"resources"`
	ModuleCalls  map[string]ModuleCall `json:"module_calls"`
}

// ConfigResource represents a resource block in the configuration.
type ConfigResource struct {
	Address     string                `json:"address"`
	Expressions map[string]Expression `json:"expressions"`
}

// ModuleCall represents a module block in the configuration
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
	Address       string   `json:"address"`
	Change        Change   `json:"change"`
	ActionReason  string   `json:"action_reason"`
}

// Change represents the details of a resource change.
type Change struct {
	Actions []string `json:"actions"`
}

// Parse executes `terraform show -json` and unmarshals the output.
func Parse(planFile string) (*TerraformPlan, error) {
	var cmd *exec.Cmd
	if planFile != "" {
		cmd = exec.Command("terraform", "show", "-json", planFile)
	} else {
		// If no plan file is provided, generate a plan and show it.
		// This requires `terraform init` to have been run.
		planCmd := exec.Command("terraform", "plan", "-out=tfplan.binary")
		if err := planCmd.Run(); err != nil {
			return nil, err
		}
		cmd = exec.Command("terraform", "show", "-json", "tfplan.binary")
	}

	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	return ParseFromData(output)
}

// ParseFromData unmarshals a Terraform plan from a byte slice.
func ParseFromData(data []byte) (*TerraformPlan, error) {
	var plan TerraformPlan
	if err := json.Unmarshal(data, &plan); err != nil {
		return nil, err
	}
	return &plan, nil
}