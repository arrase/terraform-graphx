package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "terraform-graphx [command]",
	Short: "Generate dependency graphs from Terraform infrastructure",
	Long: `terraform-graphx is a CLI tool that generates dependency graphs of your 
Terraform infrastructure and can export them to JSON, Cypher, or Neo4j.`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
