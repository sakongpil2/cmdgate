package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"fmt"
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
	bundlePath := createTempBundle(t, dir, "version: \"2.0.0\"\nmode: allowlist-only\ncommands:\n")

	tests := []struct {
		name    string
		args    []string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "validate success",
			args:    []string{"validate", "--bundle", bundlePath},
			wantErr: false,
		},
		{
			name:    "missing bundle value",
			args:    []string{"validate", "--bundle"},
			wantErr: true,
			errMsg:  "--bundle required",
		},
		{
			name:    "unknown action",
			args:    []string{"apply", "--bundle", bundlePath},
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
		})
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

// createTempBundle builds a valid policy bundle tar.gz at a temporary path.
func createTempBundle(t *testing.T, dir, allowlistContent string) string {
	t.Helper()
	manifest := []byte("version: \"2.0.0\"\ntimestamp: \"2026-06-25T00:00:00Z\"\n")
	allowlist := []byte(allowlistContent)
	sum := fmt.Sprintf("%x", sha256.Sum256(allowlist))
	checksums := []byte(sum + "\n")

	buf := &bytes.Buffer{}
	gw := gzip.NewWriter(buf)
	tw := tar.NewWriter(gw)

	files := []struct {
		name string
		body []byte
	}{
		{"manifest.yaml", manifest},
		{"allowlist.yaml", allowlist},
		{"checksums.sha256", checksums},
	}

	for _, f := range files {
		hdr := &tar.Header{
			Name: f.name,
			Mode: 0o644,
			Size: int64(len(f.body)),
		}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatalf("write header: %v", err)
		}
		if _, err := tw.Write(f.body); err != nil {
			t.Fatalf("write body: %v", err)
		}
	}

	if err := tw.Close(); err != nil {
		t.Fatalf("close tar: %v", err)
	}
	if err := gw.Close(); err != nil {
		t.Fatalf("close gzip: %v", err)
	}

	path := filepath.Join(dir, "bundle.tar.gz")
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		t.Fatalf("write bundle: %v", err)
	}
	return path
}
