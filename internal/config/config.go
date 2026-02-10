package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config represents the .emlang.yaml configuration file.
type Config struct {
	Lint    LintConfig    `yaml:"lint"`
	Diagram DiagramConfig `yaml:"diagram"`
	Fmt     FmtConfig     `yaml:"fmt"`
}

// FmtConfig holds formatter configuration.
type FmtConfig struct {
	Keys string `yaml:"keys"` // "short" or "long" (default "long")
}

// LintConfig holds linter configuration.
type LintConfig struct {
	Ignore []string `yaml:"ignore"`
}

// DiagramConfig holds diagram generation configuration.
type DiagramConfig struct {
	CSS   map[string]string `yaml:"css"`
	Serve ServeConfig       `yaml:"serve"`
}

// ServeConfig holds live-reload server configuration.
type ServeConfig struct {
	Address string `yaml:"address"`
	Port    int    `yaml:"port"`
}

// Load resolves and loads the config file with priority: flagPath > EMLANG_CONFIG env > .emlang.yaml in cwd.
// Returns a zero-value config if no file is found at the default path.
// Returns an error if an explicit path (flag or env) doesn't exist or contains invalid YAML.
func Load(flagPath string) (*Config, error) {
	path := flagPath
	explicit := true

	if path == "" {
		path = os.Getenv("EMLANG_CONFIG")
	}

	if path == "" {
		path = ".emlang.yaml"
		explicit = false
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) && !explicit {
			return &Config{}, nil
		}
		return nil, fmt.Errorf("reading config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config %s: %w", path, err)
	}

	return &cfg, nil
}
