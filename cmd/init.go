package cmd

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"terraform-graphx/internal/config"

	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize terraform-graphx configuration",
	Long:  `Initialize terraform-graphx configuration and settings.`,
}

var initConfigCmd = &cobra.Command{
	Use:   "config",
	Short: "Create a configuration file with default values",
	Long: `Create a .terraform-graphx.yaml configuration file in the current directory
with default values. You can then edit this file to set your Neo4j credentials.

The configuration file will be created with the following default values:
  - neo4j.uri: bolt://localhost:7687
  - neo4j.user: neo4j
  - neo4j.password: (empty, you must set this)

Example:
  terraform graphx init config`,
	RunE: runInitConfig,
}

func runInitConfig(cmd *cobra.Command, args []string) error {
	configPath := ".terraform-graphx.yaml"

	// Check if config file already exists
	if _, err := os.Stat(configPath); err == nil {
		return fmt.Errorf("configuration file already exists at %s", configPath)
	}

	// Create default config
	cfg := config.DefaultConfig()

	// Save to file
	if err := config.Save(cfg, configPath); err != nil {
		return fmt.Errorf("failed to create config file: %w", err)
	}

	fmt.Printf("✓ Created configuration file: %s\n\n", configPath)
	fmt.Println("Default configuration:")
	fmt.Printf("  neo4j.uri: %s\n", cfg.Neo4j.URI)
	fmt.Printf("  neo4j.user: %s\n", cfg.Neo4j.User)
	fmt.Printf("  neo4j.password: (empty)\n\n")
	fmt.Println("Please edit this file and set your Neo4j password.")

	// Attempt to update .gitignore
	if err := updateGitignore(); err != nil {
		// If gitignore update fails, print a warning but don't fail the command
		fmt.Fprintf(os.Stderr, "Warning: failed to update .gitignore: %v\n", err)
		fmt.Println("Please manually add '.terraform-graphx.yaml' and 'neo4j-data/' to your .gitignore file.")
	}

	return nil
}

// updateGitignore checks if the current directory is a git repository and if so,
// ensures that the config file and neo4j data directory are in .gitignore.
func updateGitignore() error {
	// Check if we are in a git repository
	cmd := exec.Command("git", "rev-parse", "--is-inside-work-tree")
	if err := cmd.Run(); err != nil {
		// Not a git repo, or git is not installed. Do nothing.
		fmt.Println("\nNote: Not inside a Git repository. If you initialize one later,")
		fmt.Println("remember to add '.terraform-graphx.yaml' and 'neo4j-data/' to your .gitignore")
		return nil
	}

	gitignorePath := ".gitignore"
	entriesToIgnore := []string{".terraform-graphx.yaml", "neo4j-data/"}
	var entriesAdded []string

	file, err := os.OpenFile(gitignorePath, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return fmt.Errorf("could not open or create .gitignore: %w", err)
	}
	defer file.Close()

	// Go to the beginning of the file to read it
	_, err = file.Seek(0, 0)
	if err != nil {
		return fmt.Errorf("could not seek in .gitignore: %w", err)
	}

	// Check which entries are already present
	scanner := bufio.NewScanner(file)
	existingEntries := make(map[string]bool)
	for scanner.Scan() {
		existingEntries[strings.TrimSpace(scanner.Text())] = true
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading .gitignore: %w", err)
	}

	// Append entries that are not already present
	for _, entry := range entriesToIgnore {
		if !existingEntries[entry] {
			if _, err := file.WriteString("\n" + entry); err != nil {
				return fmt.Errorf("failed to write to .gitignore: %w", err)
			}
			entriesAdded = append(entriesAdded, entry)
		}
	}

	if len(entriesAdded) > 0 {
		fmt.Printf("\n✓ Added the following entries to .gitignore: %s\n", strings.Join(entriesAdded, ", "))
	} else {
		fmt.Println("\n✓ .gitignore already contains the necessary entries.")
	}
	fmt.Println("This prevents committing sensitive credentials and local database files.")

	return nil
}

func init() {
	rootCmd.AddCommand(initCmd)
	initCmd.AddCommand(initConfigCmd)
}
