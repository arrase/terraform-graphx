package git

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// IsRepository checks if the current directory is inside a Git repository
func IsRepository() bool {
	cmd := exec.Command("git", "rev-parse", "--is-inside-work-tree")
	return cmd.Run() == nil
}

// UpdateGitignore ensures that the specified entries are present in .gitignore.
// If the current directory is not a Git repository, it prints a message and returns nil.
// Returns an error if .gitignore cannot be read or written.
func UpdateGitignore(entries []string) error {
	if !IsRepository() {
		fmt.Println("\nNote: Not inside a Git repository. If you initialize one later,")
		fmt.Printf("remember to add the following to your .gitignore: %s\n", strings.Join(entries, ", "))
		return nil
	}

	gitignorePath := ".gitignore"
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
	for _, entry := range entries {
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
