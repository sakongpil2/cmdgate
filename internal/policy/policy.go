package policy

import (
	"fmt"
	"os"

	"github.com/sakongpil2/cmdgate/internal/allowlist"
)

// ValidateAllowlistFile checks that an allowlist YAML file is parseable and
// satisfies CmdGate policy schema rules.
func ValidateAllowlistFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	cfg, err := allowlist.Parse(data)
	if err != nil {
		return fmt.Errorf("invalid allowlist.yaml: %w", err)
	}
	if err := cfg.ValidateSchema(); err != nil {
		return fmt.Errorf("invalid allowlist schema: %w", err)
	}
	return nil
}
