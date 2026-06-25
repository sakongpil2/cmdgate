package policy

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateAllowlistFile(t *testing.T) {
	path := writePolicyFile(t, "version: 1.1.0\nmode: allowlist-only\ncommands: []\n")

	if err := ValidateAllowlistFile(path); err != nil {
		t.Errorf("expected valid allowlist, got %v", err)
	}
}

func TestValidateAllowlistFile_invalidYAML(t *testing.T) {
	path := writePolicyFile(t, "[not: valid: yaml:")

	if err := ValidateAllowlistFile(path); err == nil {
		t.Error("expected error for invalid allowlist YAML, got nil")
	}
}

func TestValidateAllowlistFile_invalidSchema(t *testing.T) {
	path := writePolicyFile(t, "version: 1.1.0\nmode: allowlist-only\ncommands:\n  - cmd: \"systemctl restart kubelet\"\n")

	if err := ValidateAllowlistFile(path); err == nil {
		t.Error("expected error for invalid allowlist schema, got nil")
	}
}

func writePolicyFile(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "allowlist.yaml")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}
