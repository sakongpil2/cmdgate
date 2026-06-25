// Package allowlist parses and represents the cmdgate allowlist policy file.
package allowlist

import (
	"fmt"
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
	AllowedDirs    []string `yaml:"allowedDirs"`
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
	cmd, _, ok := c.FindCommandWithPlaceholders(argv)
	return cmd, ok
}

// Placeholder captures the type, name and matched value of a <type:name>
// placeholder token found in an allowlist command.
type Placeholder struct {
	Type  string
	Name  string
	Value string
}

// FindCommandWithPlaceholders returns the first command that matches argv and
// also extracts every <type:name> placeholder's type, name and the argv value it
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
			if typ, name, ok := PlaceholderParts(p); ok {
				placeholders = append(placeholders, Placeholder{Type: typ, Name: name, Value: argv[i]})
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

// IsPlaceholder reports whether s is a matcher placeholder token such as
// "<rpmFiles:k8s-rpms>".
func IsPlaceholder(s string) bool {
	return strings.HasPrefix(s, "<") && strings.HasSuffix(s, ">")
}

// PlaceholderParts returns the type and name parts of a placeholder token.
// For "<number:lines>" it returns ("number", "lines", true).
// For "<lines>" it returns ("", "lines", true).
// For non-placeholder tokens it returns ("", "", false).
func PlaceholderParts(s string) (typ, name string, ok bool) {
	if !IsPlaceholder(s) {
		return "", "", false
	}
	inner := strings.TrimSuffix(strings.TrimPrefix(s, "<"), ">")
	if idx := strings.Index(inner, ":"); idx >= 0 {
		return inner[:idx], inner[idx+1:], true
	}
	return "", inner, true
}

// ValidateSchema checks that the allowlist has required fields, uses supported
// matcher types, and that every placeholder references a defined matcher whose
// type matches the placeholder prefix.
func (c *Config) ValidateSchema() error {
	if c.Version == "" {
		return fmt.Errorf("allowlist version is required")
	}
	if c.Mode != "allowlist-only" {
		return fmt.Errorf("allowlist mode must be allowlist-only, got %q", c.Mode)
	}

	for i, cmd := range c.Commands {
		if cmd.ID == "" {
			return fmt.Errorf("command[%d]: id is required", i)
		}
		if cmd.Cmd == "" {
			return fmt.Errorf("command[%d]: cmd is required", i)
		}
		for _, token := range strings.Fields(cmd.Cmd) {
			if typ, name, ok := PlaceholderParts(token); ok {
				def, exists := c.Matchers[name]
				if !exists {
					return fmt.Errorf("command[%d]: unknown matcher %q", i, name)
				}
				if typ != "" && typ != def.Type {
					return fmt.Errorf("command[%d]: placeholder type %q does not match matcher type %q", i, typ, def.Type)
				}
			}
		}
	}

	for name, def := range c.Matchers {
		switch def.Type {
		case "number", "rpmFiles", "string":
		default:
			return fmt.Errorf("matcher %q: unsupported type %q", name, def.Type)
		}
	}

	return nil
}
