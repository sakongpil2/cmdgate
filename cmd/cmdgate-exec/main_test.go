package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sakongpil2/cmdgate/internal/allowlist"
)

func TestExecutorRunList(t *testing.T) {
	dir := t.TempDir()
	allowlistPath := filepath.Join(dir, "allowlist.yaml")
	auditLogPath := filepath.Join(dir, "audit.log")
	content := `
version: "1.0.0"
mode: allowlist-only
commands:
  - id: restart-kubelet
    desc: restart kubelet service
    cmd: "systemctl restart kubelet"
  - id: stop-kubelet
    desc: stop kubelet service
    cmd: "systemctl stop kubelet"
`
	if err := os.WriteFile(allowlistPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write allowlist: %v", err)
	}

	e := executor{allowlistPath: allowlistPath, auditLogPath: auditLogPath}
	if err := e.runList(); err != nil {
		t.Fatalf("runList() error = %v", err)
	}
}

func TestExecutorHandleRun(t *testing.T) {
	dir := t.TempDir()
	allowlistPath := filepath.Join(dir, "allowlist.yaml")
	auditLogPath := filepath.Join(dir, "audit.log")
	content := `
version: "1.0.0"
mode: allowlist-only
commands:
  - id: echo-hello
    desc: say hello
    cmd: "echo hello"
  - id: journalctl-lines
    desc: show kubelet logs
    cmd: "echo show <number:lines>"
matchers:
  lines:
    type: number
`
	if err := os.WriteFile(allowlistPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write allowlist: %v", err)
	}

	tests := []struct {
		name    string
		args    []string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "exact match success",
			args:    []string{"echo", "hello"},
			wantErr: false,
		},
		{
			name:    "number placeholder success",
			args:    []string{"echo", "show", "10"},
			wantErr: false,
		},
		{
			name:    "denied command",
			args:    []string{"echo", "world"},
			wantErr: true,
			errMsg:  "command not allowed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := executor{allowlistPath: allowlistPath, auditLogPath: auditLogPath}
			err := e.handleRun(tt.args)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error")
				}
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("error = %q, want containing %q", err.Error(), tt.errMsg)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestExecutorHandlePolicy(t *testing.T) {
	dir := t.TempDir()
	allowlistPath := filepath.Join(dir, "allowlist.yaml")
	auditLogPath := filepath.Join(dir, "audit.log")
	policyPath := filepath.Join(dir, "new-allowlist.yaml")
	if err := os.WriteFile(policyPath, []byte("version: \"2.0.0\"\nmode: allowlist-only\ncommands: []\n"), 0o644); err != nil {
		t.Fatalf("write policy: %v", err)
	}
	var buf strings.Builder
	oldStdout := stdout
	defer func() { stdout = oldStdout }()
	stdout = &buf

	tests := []struct {
		name    string
		args    []string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "validate success",
			args:    []string{"validate", policyPath},
			wantErr: false,
		},
		{
			name:    "legacy bundle flag rejected",
			args:    []string{"validate", "--bundle", "cmdgate-policy-1.1.0.tar.gz"},
			wantErr: true,
			errMsg:  "usage: cmdgate-exec policy validate <allowlist.yaml>",
		},
		{
			name:    "unknown action",
			args:    []string{"apply", policyPath},
			wantErr: true,
			errMsg:  "unknown policy action",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := executor{allowlistPath: allowlistPath, auditLogPath: auditLogPath}
			err := e.handlePolicy(tt.args)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error")
				}
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("error = %q, want containing %q", err.Error(), tt.errMsg)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.name == "validate success" && !strings.Contains(buf.String(), "policy valid: "+policyPath) {
				t.Errorf("success output = %q, want policy valid message", buf.String())
			}
		})
	}
}

func TestColorize(t *testing.T) {
	tests := []struct {
		name  string
		input string
		color string
		want  string
	}{
		{name: "success is green", input: "policy valid", color: ansiGreen, want: "\x1b[32mpolicy valid\x1b[0m"},
		{name: "failure is red", input: "validation failed", color: ansiRed, want: "\x1b[31mvalidation failed\x1b[0m"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := colorize(tt.input, tt.color, true)
			if got != tt.want {
				t.Errorf("colorize() = %q, want %q", got, tt.want)
			}
		})
	}
	if got := colorize("plain", ansiGreen, false); got != "plain" {
		t.Errorf("colorize disabled = %q, want plain", got)
	}
}

func TestExecutorValidatePlaceholders(t *testing.T) {
	cfg, err := allowlist.Parse([]byte(`
version: "1.0.0"
mode: allowlist-only
commands: []
matchers:
  count:
    type: number
  k8s-rpms:
    type: rpmFiles
    multiple: true
    metadataNameIn:
      - kubelet
`))
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	e := executor{}

	tests := []struct {
		name    string
		ph      []allowlist.Placeholder
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid number",
			ph:      []allowlist.Placeholder{{Name: "count", Value: "42"}},
			wantErr: false,
		},
		{
			name:    "invalid number",
			ph:      []allowlist.Placeholder{{Name: "count", Value: "abc"}},
			wantErr: true,
			errMsg:  "not a valid number",
		},
		{
			name:    "unknown matcher",
			ph:      []allowlist.Placeholder{{Name: "missing", Value: "x"}},
			wantErr: true,
			errMsg:  "unknown matcher",
		},
		{
			name:    "rpmFiles rejected at placeholder level",
			ph:      []allowlist.Placeholder{{Name: "k8s-rpms", Value: "/tmp/kubelet.rpm"}},
			wantErr: true,
			errMsg:  "rpmFiles matcher must be handled at command level",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := e.validatePlaceholders(cfg, tt.ph)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error")
				}
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("error = %q, want containing %q", err.Error(), tt.errMsg)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestExecutorHandleRunAuditUser(t *testing.T) {
	dir := t.TempDir()
	allowlistPath := filepath.Join(dir, "allowlist.yaml")
	auditLogPath := filepath.Join(dir, "audit.log")
	content := `
version: "1.0.0"
mode: allowlist-only
commands:
  - id: echo-hello
    desc: say hello
    cmd: "echo hello"
`
	if err := os.WriteFile(allowlistPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write allowlist: %v", err)
	}

	t.Setenv("SUDO_USER", "alice")
	e := executor{allowlistPath: allowlistPath, auditLogPath: auditLogPath}
	if err := e.handleRun([]string{"echo", "hello"}); err != nil {
		t.Fatalf("handleRun error: %v", err)
	}

	line, err := os.ReadFile(auditLogPath)
	if err != nil {
		t.Fatalf("read audit: %v", err)
	}
	if !strings.Contains(string(line), "user=alice") {
		t.Errorf("audit log missing SUDO_USER; got %q", string(line))
	}
}

func TestExecutorHandleAuditTail(t *testing.T) {
	dir := t.TempDir()
	auditLogPath := filepath.Join(dir, "audit.log")
	content := "line one\nline two\nline three\n"
	if err := os.WriteFile(auditLogPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write audit log: %v", err)
	}

	e := executor{auditLogPath: auditLogPath}
	if err := e.handleAudit([]string{"tail", "2"}); err != nil {
		t.Fatalf("handleAudit error: %v", err)
	}
}

func TestExecutorHandleAuditTailDefault(t *testing.T) {
	dir := t.TempDir()
	auditLogPath := filepath.Join(dir, "audit.log")
	if err := os.WriteFile(auditLogPath, []byte("line one\n"), 0o644); err != nil {
		t.Fatalf("write audit log: %v", err)
	}

	e := executor{auditLogPath: auditLogPath}
	if err := e.handleAudit([]string{"tail"}); err != nil {
		t.Fatalf("handleAudit error: %v", err)
	}
}

func TestExecutorHandleAuditTailMissingFile(t *testing.T) {
	dir := t.TempDir()
	auditLogPath := filepath.Join(dir, "audit.log")
	e := executor{auditLogPath: auditLogPath}
	if err := e.handleAudit([]string{"tail"}); err != nil {
		t.Fatalf("handleAudit error: %v", err)
	}
}

func TestExecutorHandleAuditTailInvalidCount(t *testing.T) {
	dir := t.TempDir()
	auditLogPath := filepath.Join(dir, "audit.log")
	e := executor{auditLogPath: auditLogPath}
	if err := e.handleAudit([]string{"tail", "abc"}); err == nil {
		t.Fatalf("expected error for invalid count")
	}
}

func TestPrintHelpContainsPolicyValidateYAML(t *testing.T) {
	var buf strings.Builder
	oldStdout := stdout
	defer func() { stdout = oldStdout }()
	stdout = &buf

	printHelp()
	out := buf.String()
	if !strings.Contains(out, "cmdgate-exec policy validate allowlist.yaml") {
		t.Errorf("help output missing YAML policy validate example: %q", out)
	}
	if strings.Contains(out, "--bundle") || strings.Contains(out, ".tar.gz") {
		t.Errorf("help output still contains legacy bundle usage: %q", out)
	}
	if !strings.HasSuffix(out, "\n\n") {
		t.Errorf("help output should end with a blank line: %q", out)
	}
}
