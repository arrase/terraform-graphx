package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	ConfigFileName = ".terraform-graphx"
	ConfigFileType = "yaml"
)

// Config holds the configuration for terraform-graphx.
type Config struct {
	Neo4j    Neo4jConfig `mapstructure:"neo4j"`
	Format   string      `mapstructure:"format"`
	PlanFile string      `mapstructure:"planfile"`
	Update   bool        `mapstructure:"update"`
}

// Neo4jConfig holds the Neo4j connection settings.
type Neo4jConfig struct {
	URI         string `mapstructure:"uri"`
	User        string `mapstructure:"user"`
	Password    string `mapstructure:"password"`
	DockerImage string `mapstructure:"docker_image"`
}

// DefaultConfig returns a Config with default values.
func DefaultConfig() *Config {
	return &Config{
		Neo4j: Neo4jConfig{
			URI:         "bolt://localhost:7687",
			User:        "neo4j",
			Password:    "",
			DockerImage: "neo4j:community",
		},
		Format:   "json",
		PlanFile: "",
		Update:   false,
	}
}

// Load reads the configuration from the .terraform-graphx.yaml file.
// It searches for the config file in the current directory and parent directories.
func Load() (*Config, error) {
	v := viper.New()
	v.SetConfigName(ConfigFileName)
	v.SetConfigType(ConfigFileType)

	// Add current directory and search upwards
	v.AddConfigPath(".")
	v.AddConfigPath("$HOME")

	// Set defaults
	defaults := DefaultConfig()
	v.SetDefault("neo4j.uri", defaults.Neo4j.URI)
	v.SetDefault("neo4j.user", defaults.Neo4j.User)
	v.SetDefault("neo4j.password", defaults.Neo4j.Password)

	// Read config file
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config file not found; return defaults
			return defaults, nil
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &cfg, nil
}

// LoadAndMerge loads configuration from file and merges it with CLI flags.
// Priority: flags > config file > defaults
func LoadAndMerge(cmd *cobra.Command, args []string) (*Config, error) {
	cfg, err := Load()
	if err != nil {
		return nil, err
	}

	// Override with flags
	if cmd.Flags().Changed("format") {
		cfg.Format, _ = cmd.Flags().GetString("format")
	}

	if cmd.Flags().Changed("update") {
		cfg.Update, _ = cmd.Flags().GetBool("update")
	}

	if cmd.Flags().Changed("neo4j-uri") {
		cfg.Neo4j.URI, _ = cmd.Flags().GetString("neo4j-uri")
	}

	if cmd.Flags().Changed("neo4j-user") {
		cfg.Neo4j.User, _ = cmd.Flags().GetString("neo4j-user")
	}

	if cmd.Flags().Changed("neo4j-pass") {
		cfg.Neo4j.Password, _ = cmd.Flags().GetString("neo4j-pass")
	}

	// Handle plan file from args or flag
	if len(args) > 0 {
		cfg.PlanFile = args[0]
	} else if cmd.Flags().Changed("plan") {
		cfg.PlanFile, _ = cmd.Flags().GetString("plan")
	}

	return cfg, nil
}

// Save writes the configuration to a .terraform-graphx.yaml file in the current directory.
func Save(cfg *Config, path string) error {
	if path == "" {
		path = fmt.Sprintf("%s.%s", ConfigFileName, ConfigFileType)
	}

	v := viper.New()
	v.Set("neo4j.uri", cfg.Neo4j.URI)
	v.Set("neo4j.user", cfg.Neo4j.User)
	v.Set("neo4j.password", cfg.Neo4j.Password)
	v.Set("neo4j.docker_image", cfg.Neo4j.DockerImage)

	// Ensure the directory exists
	dir := filepath.Dir(path)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create config directory: %w", err)
		}
	}

	if err := v.WriteConfigAs(path); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	// Ensure the config file is only readable/writable by the owner (contains secrets)
	if err := os.Chmod(path, 0600); err != nil {
		// Not fatal: warn the caller but return success (file was written)
		return fmt.Errorf("failed to set secure permissions on config file: %w", err)
	}

	return nil
}

// Exists checks if a config file exists in the current directory or parent directories.
func Exists() bool {
	v := viper.New()
	v.SetConfigName(ConfigFileName)
	v.SetConfigType(ConfigFileType)
	v.AddConfigPath(".")

	err := v.ReadInConfig()
	return err == nil
}
