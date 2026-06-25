// Package allowlist parses and represents the cmdgate allowlist policy file.
package allowlist

import (
	"gopkg.in/yaml.v3"
)

// Config is the top-level structure of an allowlist.yaml file.
type Config struct {
	Version  string    `yaml:"version"`
	Mode     string    `yaml:"mode"`
	Commands []Command `yaml:"commands"`
	Matchers Matchers  `yaml:"matchers"`
}

// Command describes a single allowed command in the policy.
type Command struct {
	ID   string `yaml:"id"`
	Desc string `yaml:"desc"`
	Cmd  string `yaml:"cmd"`
}

// Matchers is a map of matcher names to their definitions.
type Matchers map[string]MatcherDef

// MatcherDef defines how a placeholder value should be validated.
type MatcherDef struct {
	Type           string   `yaml:"type"`
	Multiple       bool     `yaml:"multiple"`
	MetadataNameIn []string `yaml:"metadataNameIn"`
}

// Parse parses YAML-formatted allowlist data into a Config.
func Parse(data []byte) (*Config, error) {
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
