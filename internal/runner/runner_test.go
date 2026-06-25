package runner

import (
	"testing"
)

func TestRunEcho(t *testing.T) {
	out, err := Run("echo", []string{"hello"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := string(out); got != "hello\n" {
		t.Errorf("output = %q, want hello\\n", got)
	}
}
