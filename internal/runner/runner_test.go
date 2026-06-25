package runner

import (
	"bytes"
	"errors"
	"os/exec"
	"testing"
)

func TestRunWithIOEcho(t *testing.T) {
	var stdout bytes.Buffer
	if err := RunWithIO("echo", []string{"hello"}, nil, &stdout, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := stdout.String(); got != "hello\n" {
		t.Errorf("output = %q, want hello\\n", got)
	}
}

func TestRunWithIOPropagatesExitCode(t *testing.T) {
	err := RunWithIO("false", nil, nil, nil, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("expected *exec.ExitError, got %T", err)
	}
	if exitErr.ExitCode() != 1 {
		t.Errorf("exit code = %d, want 1", exitErr.ExitCode())
	}
}
