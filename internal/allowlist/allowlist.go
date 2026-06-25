// Package allowlist parses and represents the cmdgate allowlist policy file.
package allowlist

import (
	"strings"

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

// FindCommand returns the first command in the allowlist that matches argv.
// A command matches when every fixed token equals the corresponding argv
// element and every placeholder token (<...>) matches any single value.
func (c *Config) FindCommand(argv []string) (Command, bool) {
	for _, cmd := range c.Commands {
		parts := strings.Fields(cmd.Cmd)
		if len(parts) != len(argv) {
			continue
		}
		match := true
		for i, p := range parts {
			if !isPlaceholder(p) && p != argv[i] {
				match = false
				break
			}
		}
		if match {
			return cmd, true
		}
	}
	return Command{}, false
}

// Placeholder captures the name and matched value of a <type:name>
// placeholder token found in an allowlist command.
type Placeholder struct {
	Name  string
	Value string
}

// FindCommandWithPlaceholders returns the first command that matches argv and
// also extracts every <type:name> placeholder's name and the argv value it
// matched. It behaves like FindCommand but returns placeholder metadata.
func (c *Config) FindCommandWithPlaceholders(argv []string) (Command, []Placeholder, bool) {
	for _, cmd := range c.Commands {
		parts := strings.Fields(cmd.Cmd)
		if len(parts) != len(argv) {
			continue
		}
		var placeholders []Placeholder
		match := true
		for i, p := range parts {
			if isPlaceholder(p) {
				name := strings.TrimSuffix(strings.TrimPrefix(p, "<"), ">")
				if idx := strings.Index(name, ":"); idx >= 0 {
					name = name[idx+1:]
				}
				placeholders = append(placeholders, Placeholder{Name: name, Value: argv[i]})
				continue
			}
			if p != argv[i] {
				match = false
				break
			}
		}
		if match {
			return cmd, placeholders, true
		}
	}
	return Command{}, nil, false
}

// isPlaceholder reports whether s is a matcher placeholder token such as
// "<rpmFiles:k8s-rpms>".
func isPlaceholder(s string) bool {
	return strings.HasPrefix(s, "<") && strings.HasSuffix(s, ">")
}
