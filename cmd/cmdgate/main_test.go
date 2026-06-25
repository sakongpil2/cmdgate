package main

import (
	"testing"
)

func TestBuildExecCommand(t *testing.T) {
	args := buildExecCommand([]string{"run", "systemctl", "status", "kubelet"})
	want := []string{"-n", "/opt/cmdgate/cmdgate-exec", "run", "systemctl", "status", "kubelet"}
	if len(args) != len(want) {
		t.Fatalf("len(args) = %d, want %d", len(args), len(want))
	}
	for i := range want {
		if args[i] != want[i] {
			t.Errorf("args[%d] = %q, want %q", i, args[i], want[i])
		}
	}
}

func TestBuildExecCommandNoArgs(t *testing.T) {
	args := buildExecCommand(nil)
	if len(args) != 2 || args[0] != "-n" || args[1] != execBinary {
		t.Errorf("args = %v", args)
	}
}
