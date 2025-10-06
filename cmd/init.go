package cmd

import (
	"bufio"
	"crypto/rand"
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
	Long: `Initialize terraform-graphx configuration and settings.

Creates a .terraform-graphx.yaml configuration file in the current directory
with default values and a randomly generated password. Also creates the neo4j-data
directory for Docker volume mounting.

The configuration file will be created with the following default values:
  - neo4j.uri: bolt://localhost:7687
  - neo4j.user: neo4j
  - neo4j.password: (randomly generated)
  - neo4j.docker_image: neo4j:community

Example:
  terraform-graphx init`,
	RunE: runInit,
}

func runInit(cmd *cobra.Command, args []string) error {
	configPath := ".terraform-graphx.yaml"

	// Check if config file already exists
	if _, err := os.Stat(configPath); err == nil {
		return fmt.Errorf("configuration file already exists at %s", configPath)
	}

	// Create default config
	cfg := config.DefaultConfig()

	// Generate random password
	password, err := generateRandomPassword(16)
	if err != nil {
		return fmt.Errorf("failed to generate random password: %w", err)
	}
	cfg.Neo4j.Password = password

	// Save to file
	if err := config.Save(cfg, configPath); err != nil {
		return fmt.Errorf("failed to create config file: %w", err)
	}

	// Create neo4j-data directory
	dataDir := "neo4j-data"
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return fmt.Errorf("failed to create neo4j-data directory: %w", err)
	}

	fmt.Printf("✓ Created configuration file: %s\n\n", configPath)
	fmt.Println("Default configuration:")
	fmt.Printf("  neo4j.uri: %s\n", cfg.Neo4j.URI)
	fmt.Printf("  neo4j.user: %s\n", cfg.Neo4j.User)
	fmt.Printf("  neo4j.password: %s\n", cfg.Neo4j.Password)
	fmt.Printf("  neo4j.docker_image: %s\n\n", cfg.Neo4j.DockerImage)
	fmt.Printf("✓ Created data directory: %s\n\n", dataDir)

	// Attempt to update .gitignore
	if err := updateGitignore(); err != nil {
		// If gitignore update fails, print a warning but don't fail the command
		fmt.Fprintf(os.Stderr, "Warning: failed to update .gitignore: %v\n", err)
		fmt.Println("Please manually add '.terraform-graphx.yaml' and 'neo4j-data/' to your .gitignore file.")
	}

	return nil
}

// generateRandomPassword generates a random alphanumeric password of the specified length
func generateRandomPassword(length int) (string, error) {
	// Use only alphanumeric characters to avoid issues with special characters in Neo4j auth string
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	for i := range bytes {
		bytes[i] = charset[int(bytes[i])%len(charset)]
	}
	return string(bytes), nil
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
}
