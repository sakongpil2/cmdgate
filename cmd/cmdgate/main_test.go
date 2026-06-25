package main

import (
	"strings"
	"testing"
)

func TestBuildExecCommand(t *testing.T) {
	got := buildExecCommand([]string{"run", "list"})
	want := []string{"-n", "/opt/cmdgate/cmdgate-exec", "run", "list"}
	if len(got) != len(want) {
		t.Fatalf("len = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestPrintHelpContainsCommands(t *testing.T) {
	var buf strings.Builder
	oldStdout := stdout
	defer func() { stdout = oldStdout }()
	stdout = &buf
	printHelp()
	out := buf.String()
	for _, cmd := range []string{"run", "policy", "audit", "help"} {
		if !strings.Contains(out, cmd) {
			t.Errorf("help output missing %q", cmd)
		}
	}
}
