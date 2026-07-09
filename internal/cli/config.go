package cli

import (
	"fmt"
	"os"

	"github.com/pelletier/go-toml/v2"
)

// Rule defines a pattern-based routing rule.
type Rule struct {
	Pattern string `toml:"pattern"`
	Prefer  string `toml:"prefer"`
}

// Config represents the user's stamp configuration.
type Config struct {
	Precedence []string `toml:"precedence"`
	Rules      []Rule   `toml:"rules"`
}

// LoadConfig reads and parses the config.toml file.
// Returns default values if the file does not exist.
func LoadConfig(path string) (*Config, error) {
	cfg := &Config{
		Precedence: []string{"dnf", "flatpak", "brew"},
	}

	//nolint:gosec // path is resolved internally via XDG config dir
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	if err := toml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}
	return cfg, nil
}
