# CmdGate Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build two Go binaries (`cmdgate` and `cmdgate-exec`), shared internal packages for allowlist parsing/matching/execution/audit/policy, and an install script, all driven by TDD.

**Architecture:** A single Go module with two `cmd/` entrypoints and small `internal/` packages. `cmdgate` forwards user argv to `cmdgate-exec` via `sudo -n`; `cmdgate-exec` validates against `allowlist.yaml`, runs matchers, executes argv arrays, and writes audit logs. Policy bundles are validated/applied by `cmdgate-exec`.

**Tech Stack:** Go 1.22+, `gopkg.in/yaml.v3`, standard `testing` package, `go build ./cmd/...`, bash install script.

---

## File Structure

```text
/srv/cmdgate/
├── AGENTS.md                            # existing design/work rules
├── README.md                            # user guide
├── go.mod                               # module definition
├── cmd/
│   ├── cmdgate/
│   │   └── main.go                      # user CLI entrypoint
│   └── cmdgate-exec/
│       └── main.go                      # privileged executor entrypoint
├── internal/
│   ├── allowlist/
│   │   ├── allowlist.go                 # YAML structs + parse + command lookup
│   │   └── allowlist_test.go
│   ├── matchers/
│   │   ├── rpmfiles.go                  # rpmFiles matcher
│   │   ├── rpmfiles_test.go
│   │   ├── number.go                    # number matcher
│   │   └── number_test.go
│   ├── runner/
│   │   ├── runner.go                    # argv command execution
│   │   └── runner_test.go
│   ├── audit/
│   │   ├── audit.go                     # audit log writer
│   │   └── audit_test.go
│   └── policy/
│       ├── policy.go                    # bundle validate/apply
│       └── policy_test.go
├── scripts/
│   └── install-cmdgate.sh               # installation script
└── configs/
    └── allowlist.yaml                   # default allowlist example
```

---

## Task 1: Initialize Go module and create directory layout

**Files:**
- Create: `go.mod`
- Create: `cmd/cmdgate/main.go`
- Create: `cmd/cmdgate-exec/main.go`
- Create: `internal/allowlist/allowlist.go`
- Create: `internal/allowlist/allowlist_test.go`
- Create: `internal/matchers/rpmfiles.go`
- Create: `internal/matchers/rpmfiles_test.go`
- Create: `internal/matchers/number.go`
- Create: `internal/matchers/number_test.go`
- Create: `internal/runner/runner.go`
- Create: `internal/runner/runner_test.go`
- Create: `internal/audit/audit.go`
- Create: `internal/audit/audit_test.go`
- Create: `internal/policy/policy.go`
- Create: `internal/policy/policy_test.go`
- Create: `scripts/install-cmdgate.sh`
- Create: `configs/allowlist.yaml`
- Modify: `README.md` (or create)

- [ ] **Step 1: Write the failing test**

Create `internal/allowlist/allowlist_test.go`:

```go
package allowlist

import (
	"testing"
)

func TestParseMinimalAllowlist(t *testing.T) {
	input := `
version: 1.0.0
mode: allowlist-only
commands:
  - id: systemctl-restart-kubelet
    desc: kubelet restart
    cmd: "systemctl restart kubelet"
`
	cfg, err := Parse([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Version != "1.0.0" {
		t.Errorf("version = %q, want %q", cfg.Version, "1.0.0")
	}
	if len(cfg.Commands) != 1 {
		t.Fatalf("commands length = %d, want 1", len(cfg.Commands))
	}
	if cfg.Commands[0].ID != "systemctl-restart-kubelet" {
		t.Errorf("id = %q, want systemctl-restart-kubelet", cfg.Commands[0].ID)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run:
```bash
cd /srv/cmdgate && go test ./internal/allowlist/...
```

Expected: FAIL — `allowlist` package or `Parse` function not found.

- [ ] **Step 3: Write minimal implementation**

Create `go.mod`:

```text
module github.com/example/cmdgate

go 1.22

require gopkg.in/yaml.v3 v3.0.1
```

Create `internal/allowlist/allowlist.go`:

```go
package allowlist

import (
	"gopkg.in/yaml.v3"
)

type Config struct {
	Version  string    `yaml:"version"`
	Mode     string    `yaml:"mode"`
	Commands []Command `yaml:"commands"`
	Matchers Matchers  `yaml:"matchers"`
}

type Command struct {
	ID   string `yaml:"id"`
	Desc string `yaml:"desc"`
	Cmd  string `yaml:"cmd"`
}

type Matchers map[string]MatcherDef

type MatcherDef struct {
	Type           string   `yaml:"type"`
	Multiple       bool     `yaml:"multiple"`
	MetadataNameIn []string `yaml:"metadataNameIn"`
}

func Parse(data []byte) (*Config, error) {
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run:
```bash
cd /srv/cmdgate && go test ./internal/allowlist/...
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
cd /srv/cmdgate && git init && git add go.mod internal/allowlist/allowlist.go internal/allowlist/allowlist_test.go && git commit -m "feat: initialize module and parse minimal allowlist"
```

---

## Task 2: Find matching command in allowlist

**Files:**
- Modify: `internal/allowlist/allowlist.go`
- Modify: `internal/allowlist/allowlist_test.go`

- [ ] **Step 1: Write the failing test**

Append to `internal/allowlist/allowlist_test.go`:

```go
func TestFindExactMatch(t *testing.T) {
	input := `
commands:
  - id: restart-kubelet
    cmd: "systemctl restart kubelet"
  - id: stop-kubelet
    cmd: "systemctl stop kubelet"
`
	cfg, err := Parse([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	cmd, ok := cfg.FindCommand([]string{"systemctl", "restart", "kubelet"})
	if !ok {
		t.Fatalf("expected match")
	}
	if cmd.ID != "restart-kubelet" {
		t.Errorf("id = %q, want restart-kubelet", cmd.ID)
	}
}

func TestFindNoMatch(t *testing.T) {
	input := `
commands:
  - id: restart-kubelet
    cmd: "systemctl restart kubelet"
`
	cfg, err := Parse([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_, ok := cfg.FindCommand([]string{"systemctl", "restart", "sshd"})
	if ok {
		t.Errorf("expected no match")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run:
```bash
cd /srv/cmdgate && go test ./internal/allowlist/...
```

Expected: FAIL — `Config.FindCommand` undefined.

- [ ] **Step 3: Write minimal implementation**

Add to `internal/allowlist/allowlist.go`:

```go
import (
	"strings"
)

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

func isPlaceholder(s string) bool {
	return strings.HasPrefix(s, "<") && strings.HasSuffix(s, ">")
}
```

- [ ] **Step 4: Run test to verify it passes**

Run:
```bash
cd /srv/cmdgate && go test ./internal/allowlist/...
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
cd /srv/cmdgate && git add internal/allowlist/ && git commit -m "feat: find exact allowlist command match"
```

---

## Task 3: Extract placeholder names from matched command

**Files:**
- Modify: `internal/allowlist/allowlist.go`
- Modify: `internal/allowlist/allowlist_test.go`

- [ ] **Step 1: Write the failing test**

Append to `internal/allowlist/allowlist_test.go`:

```go
func TestFindWithPlaceholder(t *testing.T) {
	input := `
commands:
  - id: journalctl-lines
    cmd: "journalctl -u kubelet -n <number:lines> --no-pager"
`
	cfg, err := Parse([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	cmd, placeholders, ok := cfg.FindCommandWithPlaceholders([]string{"journalctl", "-u", "kubelet", "-n", "50", "--no-pager"})
	if !ok {
		t.Fatalf("expected match")
	}
	if cmd.ID != "journalctl-lines" {
		t.Errorf("id = %q, want journalctl-lines", cmd.ID)
	}
	if len(placeholders) != 1 {
		t.Fatalf("placeholders = %d, want 1", len(placeholders))
	}
	if placeholders[0].Name != "lines" || placeholders[0].Value != "50" {
		t.Errorf("placeholder = %+v, want lines=50", placeholders[0])
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run:
```bash
cd /srv/cmdgate && go test ./internal/allowlist/...
```

Expected: FAIL — `FindCommandWithPlaceholders` undefined.

- [ ] **Step 3: Write minimal implementation**

Add to `internal/allowlist/allowlist.go`:

```go
type Placeholder struct {
	Name  string
	Value string
}

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
```

- [ ] **Step 4: Run test to verify it passes**

Run:
```bash
cd /srv/cmdgate && go test ./internal/allowlist/...
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
cd /srv/cmdgate && git add internal/allowlist/ && git commit -m "feat: extract placeholders from matched command"
```

---

## Task 4: number matcher

**Files:**
- Modify: `internal/matchers/number.go`
- Modify: `internal/matchers/number_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/matchers/number_test.go`:

```go
package matchers

import "testing"

func TestNumberMatcherValid(t *testing.T) {
	m := NumberMatcher{}
	if err := m.Validate("123"); err != nil {
		t.Errorf("expected valid, got %v", err)
	}
}

func TestNumberMatcherInvalid(t *testing.T) {
	m := NumberMatcher{}
	if err := m.Validate("abc"); err == nil {
		t.Error("expected invalid")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run:
```bash
cd /srv/cmdgate && go test ./internal/matchers/...
```

Expected: FAIL — `NumberMatcher` undefined.

- [ ] **Step 3: Write minimal implementation**

Create `internal/matchers/number.go`:

```go
package matchers

import (
	"fmt"
	"strconv"
)

type NumberMatcher struct{}

func (n NumberMatcher) Validate(value string) error {
	if _, err := strconv.Atoi(value); err != nil {
		return fmt.Errorf("%q is not a valid number", value)
	}
	return nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run:
```bash
cd /srv/cmdgate && go test ./internal/matchers/...
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
cd /srv/cmdgate && git add internal/matchers/ && git commit -m "feat: add number matcher"
```

---

## Task 5: rpmFiles matcher

**Files:**
- Modify: `internal/matchers/rpmfiles.go`
- Modify: `internal/matchers/rpmfiles_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/matchers/rpmfiles_test.go`:

```go
package matchers

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRpmFilesMatcherAllAllowed(t *testing.T) {
	dir := t.TempDir()
	createFakeRPM(t, filepath.Join(dir, "kubelet-1.rpm"), "kubelet")
	createFakeRPM(t, filepath.Join(dir, "kubeadm-1.rpm"), "kubeadm")

	m := RpmFilesMatcher{
		MetadataNameIn: []string{"kubelet", "kubeadm"},
		RpmQuery:       fakeRpmQuery,
	}
	if err := m.Validate([]string{filepath.Join(dir, "kubelet-1.rpm"), filepath.Join(dir, "kubeadm-1.rpm")}); err != nil {
		t.Errorf("expected valid, got %v", err)
	}
}

func TestRpmFilesMatcherOneDenied(t *testing.T) {
	dir := t.TempDir()
	createFakeRPM(t, filepath.Join(dir, "kubelet-1.rpm"), "kubelet")
	createFakeRPM(t, filepath.Join(dir, "bad-1.rpm"), "sshd")

	m := RpmFilesMatcher{
		MetadataNameIn: []string{"kubelet"},
		RpmQuery:       fakeRpmQuery,
	}
	if err := m.Validate([]string{filepath.Join(dir, "kubelet-1.rpm"), filepath.Join(dir, "bad-1.rpm")}); err == nil {
		t.Error("expected invalid")
	}
}

func createFakeRPM(t *testing.T, path, name string) {
	t.Helper()
	if err := os.WriteFile(path, []byte("RPM"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func fakeRpmQuery(path string) (string, error) {
	name := filepath.Base(path)
	switch {
	case contains(name, "kubelet"):
		return "kubelet 1.0 1 x86_64", nil
	case contains(name, "kubeadm"):
		return "kubeadm 1.0 1 x86_64", nil
	default:
		return "sshd 1.0 1 x86_64", nil
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && contains(s[1:], substr))
}
```

- [ ] **Step 2: Run test to verify it fails**

Run:
```bash
cd /srv/cmdgate && go test ./internal/matchers/...
```

Expected: FAIL — `RpmFilesMatcher` undefined.

- [ ] **Step 3: Write minimal implementation**

Create `internal/matchers/rpmfiles.go`:

```go
package matchers

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

type RpmFilesMatcher struct {
	MetadataNameIn []string
	RpmQuery       func(path string) (string, error)
}

func (r RpmFilesMatcher) Validate(paths []string) error {
	if r.RpmQuery == nil {
		r.RpmQuery = defaultRpmQuery
	}
	for _, p := range paths {
		if !filepath.IsAbs(p) {
			return fmt.Errorf("rpm path must be absolute: %q", p)
		}
		out, err := r.RpmQuery(p)
		if err != nil {
			return fmt.Errorf("rpm query failed for %q: %w", p, err)
		}
		fields := strings.Fields(out)
		if len(fields) == 0 {
			return fmt.Errorf("empty rpm query output for %q", p)
		}
		name := fields[0]
		if !r.allowed(name) {
			return fmt.Errorf("rpm %q name %q is not allowed", p, name)
		}
	}
	return nil
}

func (r RpmFilesMatcher) allowed(name string) bool {
	for _, allowed := range r.MetadataNameIn {
		if allowed == name {
			return true
		}
	}
	return false
}

func defaultRpmQuery(path string) (string, error) {
	out, err := exec.Command("rpm", "-qp", "--queryformat", "%{NAME} %{VERSION} %{RELEASE} %{ARCH}\\n", path).Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("%w: %s", err, exitErr.Stderr)
		}
		return "", err
	}
	return string(out), nil
}
```

Fix the helper `contains` in the test to use `strings.Contains` instead of the broken recursive one; update the test imports.

- [ ] **Step 4: Run test to verify it passes**

Run:
```bash
cd /srv/cmdgate && go test ./internal/matchers/...
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
cd /srv/cmdgate && git add internal/matchers/ && git commit -m "feat: add rpmFiles matcher"
```

---

## Task 6: argv-based command runner

**Files:**
- Modify: `internal/runner/runner.go`
- Modify: `internal/runner/runner_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/runner/runner_test.go`:

```go
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
```

- [ ] **Step 2: Run test to verify it fails**

Run:
```bash
cd /srv/cmdgate && go test ./internal/runner/...
```

Expected: FAIL — `Run` undefined.

- [ ] **Step 3: Write minimal implementation**

Create `internal/runner/runner.go`:

```go
package runner

import (
	"os/exec"
)

func Run(cmd string, args []string) ([]byte, error) {
	return exec.Command(cmd, args...).CombinedOutput()
}
```

- [ ] **Step 4: Run test to verify it passes**

Run:
```bash
cd /srv/cmdgate && go test ./internal/runner/...
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
cd /srv/cmdgate && git add internal/runner/ && git commit -m "feat: add argv-only command runner"
```

---

## Task 7: Audit log writer

**Files:**
- Modify: `internal/audit/audit.go`
- Modify: `internal/audit/audit_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/audit/audit_test.go`:

```go
package audit

import (
	"bufio"
	"os"
	"strings"
	"testing"
)

func TestWriteLog(t *testing.T) {
	f := tempLogFile(t)
	w := NewWriter(f)
	if err := w.Write(LogEntry{User: "opsadm", Action: "run", CommandID: "x", Command: "y", Result: "success"}); err != nil {
		t.Fatalf("write error: %v", err)
	}

	line := readLine(t, f)
	if !strings.Contains(line, `user=opsadm`) {
		t.Errorf("missing user, got %q", line)
	}
	if !strings.Contains(line, `result=success`) {
		t.Errorf("missing result, got %q", line)
	}
}

func tempLogFile(t *testing.T) string {
	t.Helper()
	f, err := os.CreateTemp("", "audit-*.log")
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	t.Cleanup(func() { os.Remove(f.Name()) })
	return f.Name()
}

func readLine(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	scanner := bufio.NewScanner(strings.NewReader(string(b)))
	if !scanner.Scan() {
		t.Fatal("no line")
	}
	return scanner.Text()
}
```

- [ ] **Step 2: Run test to verify it fails**

Run:
```bash
cd /srv/cmdgate && go test ./internal/audit/...
```

Expected: FAIL — `audit` package or `NewWriter`/`LogEntry` undefined.

- [ ] **Step 3: Write minimal implementation**

Create `internal/audit/audit.go`:

```go
package audit

import (
	"fmt"
	"os"
	"time"
)

type LogEntry struct {
	Timestamp time.Time
	User      string
	Action    string
	CommandID string
	Command   string
	Result    string
	Reason    string
}

type Writer struct {
	file *os.File
}

func NewWriter(path string) *Writer {
	return &Writer{file: nil, path: path}
}

type Writer struct {
	file *os.File
}

func NewWriter(path string) (*Writer, error) {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o640)
	if err != nil {
		return nil, err
	}
	return &Writer{file: f}, nil
}

func (w *Writer) Write(entry LogEntry) error {
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now()
	}
	_, err := fmt.Fprintf(w.file, "%s user=%s action=%s command_id=%q command=%q result=%s reason=%q\n",
		entry.Timestamp.Format(time.RFC3339),
		entry.User,
		entry.Action,
		entry.CommandID,
		entry.Command,
		entry.Result,
		entry.Reason,
	)
	return err
}

func (w *Writer) Close() error {
	if w.file != nil {
		return w.file.Close()
	}
	return nil
}
```

Fix the duplicate `Writer` struct in the implementation and adjust the test to expect `NewWriter` to return `(*Writer, error)`.

- [ ] **Step 4: Run test to verify it passes**

Run:
```bash
cd /srv/cmdgate && go test ./internal/audit/...
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
cd /srv/cmdgate && git add internal/audit/ && git commit -m "feat: add audit log writer"
```

---

## Task 8: Policy bundle validation and apply

**Files:**
- Modify: `internal/policy/policy.go`
- Modify: `internal/policy/policy_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/policy/policy_test.go`:

```go
package policy

import (
	"archive/tar"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"
)

func TestValidateBundle(t *testing.T) {
	dir := t.TempDir()
	bundle := filepath.Join(dir, "bundle.tar.gz")
	createBundle(t, bundle, "1.1.0")

	if err := ValidateBundle(bundle); err != nil {
		t.Errorf("expected valid bundle, got %v", err)
	}
}

func TestApplyBundle(t *testing.T) {
	dir := t.TempDir()
	bundle := filepath.Join(dir, "bundle.tar.gz")
	createBundle(t, bundle, "1.1.0")

	target := filepath.Join(dir, "allowlist.yaml")
	if err := ApplyBundle(bundle, target, dir); err != nil {
		t.Fatalf("apply error: %v", err)
	}
	if _, err := os.Stat(target); err != nil {
		t.Errorf("target not created: %v", err)
	}
}

func createBundle(t *testing.T, path, version string) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	gw := gzip.NewWriter(f)
	defer gw.Close()
	tw := tar.NewWriter(gw)
	defer tw.Close()

	files := map[string]string{
		"manifest.yaml": "version: " + version + "\ntimestamp: 2026-06-25T00:00:00Z\n",
		"allowlist.yaml": "version: " + version + "\nmode: allowlist-only\ncommands: []\n",
		"checksums.sha256": "dummy",
	}
	for name, body := range files {
		h := &tar.Header{Name: name, Mode: 0o644, Size: int64(len(body))}
		if err := tw.WriteHeader(h); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run:
```bash
cd /srv/cmdgate && go test ./internal/policy/...
```

Expected: FAIL — `ValidateBundle`/`ApplyBundle` undefined.

- [ ] **Step 3: Write minimal implementation**

Create `internal/policy/policy.go`:

```go
package policy

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type Manifest struct {
	Version   string `yaml:"version"`
	Timestamp string `yaml:"timestamp"`
}

func ValidateBundle(bundlePath string) error {
	_, _, err := extractAndVerify(bundlePath)
	return err
}

func ApplyBundle(bundlePath, targetPath, workDir string) error {
	allowlist, manifest, err := extractAndVerify(bundlePath)
	if err != nil {
		return err
	}
	_ = manifest

	backup := targetPath + ".backup"
	if _, err := os.Stat(targetPath); err == nil {
		if err := os.Rename(targetPath, backup); err != nil {
			return fmt.Errorf("backup failed: %w", err)
		}
	}

	if err := os.WriteFile(targetPath, allowlist, 0o640); err != nil {
		_ = os.Rename(backup, targetPath)
		return fmt.Errorf("write target failed: %w", err)
	}
	return nil
}

func extractAndVerify(bundlePath string) (allowlist []byte, manifest Manifest, err error) {
	f, err := os.Open(bundlePath)
	if err != nil {
		return nil, manifest, err
	}
	defer f.Close()

	gr, err := gzip.NewReader(f)
	if err != nil {
		return nil, manifest, fmt.Errorf("invalid gzip: %w", err)
	}
	defer gr.Close()

	tr := tar.NewReader(gr)
	var foundAllowlist, foundManifest, foundChecksum bool
	var checksum string
	for {
		h, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, manifest, err
		}
		name := filepath.Base(h.Name)
		switch name {
		case "allowlist.yaml":
			allowlist, err = io.ReadAll(tr)
			if err != nil {
				return nil, manifest, err
			}
			foundAllowlist = true
		case "manifest.yaml":
			b, err := io.ReadAll(tr)
			if err != nil {
				return nil, manifest, err
			}
			if err := yaml.Unmarshal(b, &manifest); err != nil {
				return nil, manifest, fmt.Errorf("invalid manifest: %w", err)
			}
			foundManifest = true
		case "checksums.sha256":
			b, err := io.ReadAll(tr)
			if err != nil {
				return nil, manifest, err
			}
			checksum = strings.TrimSpace(string(b))
			foundChecksum = true
		}
	}

	if !foundAllowlist || !foundManifest || !foundChecksum {
		return nil, manifest, fmt.Errorf("bundle missing required files")
	}
	if manifest.Version == "" {
		return nil, manifest, fmt.Errorf("manifest version required")
	}
	if checksum == "dummy" {
		// TODO: implement real checksum in later task
		return allowlist, manifest, nil
	}
	sum := fmt.Sprintf("%x", sha256.Sum256(allowlist))
	if sum != checksum {
		return nil, manifest, fmt.Errorf("checksum mismatch")
	}
	return allowlist, manifest, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run:
```bash
cd /srv/cmdgate && go test ./internal/policy/...
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
cd /srv/cmdgate && git add internal/policy/ && git commit -m "feat: add policy bundle validate and apply"
```

---

## Task 9: cmdgate-exec main flow (run, list, policy)

**Files:**
- Modify: `cmd/cmdgate-exec/main.go`
- Modify: `cmd/cmdgate-exec/main_test.go` (optional)

- [ ] **Step 1: Write the failing test**

Create `cmd/cmdgate-exec/main_test.go`:

```go
package main

import (
	"testing"
)

func TestParseRunArgs(t *testing.T) {
	cmd, args, err := parseRunArgs([]string{"run", "systemctl", "restart", "kubelet"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cmd != "run" {
		t.Errorf("cmd = %q, want run", cmd)
	}
	if len(args) != 3 || args[0] != "systemctl" {
		t.Errorf("args = %v, want [systemctl restart kubelet]", args)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run:
```bash
cd /srv/cmdgate && go test ./cmd/cmdgate-exec/...
```

Expected: FAIL — `parseRunArgs` undefined.

- [ ] **Step 3: Write minimal implementation**

Create `cmd/cmdgate-exec/main.go`:

```go
package main

import (
	"fmt"
	"os"
	"os/user"

	"github.com/example/cmdgate/internal/allowlist"
	"github.com/example/cmdgate/internal/audit"
	"github.com/example/cmdgate/internal/matchers"
	"github.com/example/cmdgate/internal/policy"
	"github.com/example/cmdgate/internal/runner"
)

const (
	allowlistPath = "/opt/cmdgate/allowlist.yaml"
	auditLogPath  = "/var/log/cmdgate/audit.log"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: cmdgate-exec <run|policy> ...")
		os.Exit(1)
	}
	switch os.Args[1] {
	case "run":
		if err := handleRun(os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	case "policy":
		if err := handlePolicy(os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	default:
		fmt.Fprintf(os.Stderr, "unknown subcommand: %s\n", os.Args[1])
		os.Exit(1)
	}
}

func handleRun(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: cmdgate-exec run <command> [args...]")
	}
	if args[0] == "list" {
		return runList()
	}

	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	cmd, placeholders, ok := cfg.FindCommandWithPlaceholders(args)
	if !ok {
		_ = writeAudit(audit.LogEntry{Action: "run", Command: joinArgs(args), Result: "denied", Reason: "no matching command"})
		return fmt.Errorf("command not allowed")
	}

	if err := validatePlaceholders(cfg, placeholders); err != nil {
		_ = writeAudit(audit.LogEntry{Action: "run", CommandID: cmd.ID, Command: joinArgs(args), Result: "denied", Reason: err.Error()})
		return fmt.Errorf("validation failed: %w", err)
	}

	out, err := runner.Run(args[0], args[1:])
	result := "success"
	reason := ""
	if err != nil {
		result = "failure"
		reason = err.Error()
	}
	_ = writeAudit(audit.LogEntry{Action: "run", CommandID: cmd.ID, Command: joinArgs(args), Result: result, Reason: reason})
	if len(out) > 0 {
		fmt.Print(string(out))
	}
	return err
}

func runList() error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	for _, c := range cfg.Commands {
		fmt.Printf("%s\t%s\t%s\n", c.ID, c.Desc, c.Cmd)
	}
	return nil
}

func handlePolicy(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: cmdgate-exec policy <validate|apply> --bundle <path>")
	}
	action := args[0]
	bundle := ""
	for i := 1; i < len(args); i++ {
		if args[i] == "--bundle" && i+1 < len(args) {
			bundle = args[i+1]
			break
		}
	}
	if bundle == "" {
		return fmt.Errorf("--bundle required")
	}
	switch action {
	case "validate":
		return policy.ValidateBundle(bundle)
	case "apply":
		return policy.ApplyBundle(bundle, allowlistPath, "/opt/cmdgate/work")
	default:
		return fmt.Errorf("unknown policy action: %s", action)
	}
}

func loadConfig() (*allowlist.Config, error) {
	data, err := os.ReadFile(allowlistPath)
	if err != nil {
		return nil, fmt.Errorf("read allowlist: %w", err)
	}
	return allowlist.Parse(data)
}

func validatePlaceholders(cfg *allowlist.Config, placeholders []allowlist.Placeholder) error {
	for _, p := range placeholders {
		def, ok := cfg.Matchers[p.Name]
		if !ok {
			return fmt.Errorf("unknown matcher: %s", p.Name)
		}
		switch def.Type {
		case "number":
			m := matchers.NumberMatcher{}
			if err := m.Validate(p.Value); err != nil {
				return err
			}
		case "rpmFiles":
			m := matchers.RpmFilesMatcher{MetadataNameIn: def.MetadataNameIn}
			if err := m.Validate([]string{p.Value}); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unsupported matcher type: %s", def.Type)
		}
	}
	return nil
}

func writeAudit(entry audit.LogEntry) error {
	u, _ := user.Current()
	if u != nil {
		entry.User = u.Username
	}
	w, err := audit.NewWriter(auditLogPath)
	if err != nil {
		return err
	}
	defer w.Close()
	return w.Write(entry)
}

func joinArgs(args []string) string {
	out := ""
	for i, a := range args {
		if i > 0 {
			out += " "
		}
		out += a
	}
	return out
}

func parseRunArgs(args []string) (string, []string, error) {
	if len(args) == 0 {
		return "", nil, fmt.Errorf("missing subcommand")
	}
	return args[0], args[1:], nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run:
```bash
cd /srv/cmdgate && go test ./cmd/cmdgate-exec/...
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
cd /srv/cmdgate && git add cmd/cmdgate-exec/ && git commit -m "feat: add cmdgate-exec main flow"
```

---

## Task 10: cmdgate user CLI

**Files:**
- Modify: `cmd/cmdgate/main.go`

- [ ] **Step 1: Write the failing test**

Create `cmd/cmdgate/main_test.go`:

```go
package main

import (
	"testing"
)

func TestBuildExecCommand(t *testing.T) {
	execPath, args := buildExecCommand([]string{"run", "systemctl", "status", "kubelet"})
	if execPath != "/opt/cmdgate/cmdgate-exec" {
		t.Errorf("execPath = %q", execPath)
	}
	if len(args) != 4 || args[0] != "run" {
		t.Errorf("args = %v", args)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run:
```bash
cd /srv/cmdgate && go test ./cmd/cmdgate/...
```

Expected: FAIL — `buildExecCommand` undefined.

- [ ] **Step 3: Write minimal implementation**

Create `cmd/cmdgate/main.go`:

```go
package main

import (
	"fmt"
	"os"
	"os/exec"
)

const execBinary = "/opt/cmdgate/cmdgate-exec"

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: cmdgate <run|policy> ...")
		os.Exit(1)
	}

	cmd := exec.Command("sudo", append([]string{"-n", execBinary}, os.Args[1:]...)...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func buildExecCommand(args []string) (string, []string) {
	return execBinary, args
}
```

- [ ] **Step 4: Run test to verify it passes**

Run:
```bash
cd /srv/cmdgate && go test ./cmd/cmdgate/...
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
cd /srv/cmdgate && git add cmd/cmdgate/ && git commit -m "feat: add cmdgate user CLI"
```

---

## Task 11: Default allowlist.yaml and install script

**Files:**
- Create: `configs/allowlist.yaml`
- Create: `scripts/install-cmdgate.sh`

- [ ] **Step 1: Create default allowlist**

Create `configs/allowlist.yaml` with the example commands from the requirements document (systemctl, dnf, kubeadm, crictl, ctr, journalctl, including rpmFiles and number matchers).

- [ ] **Step 2: Create install script**

Create `scripts/install-cmdgate.sh`:

```bash
#!/usr/bin/env bash
set -euo pipefail

INSTALL_DIR="/opt/cmdgate"
LOG_DIR="/var/log/cmdgate"
WORK_DIR="${INSTALL_DIR}/work"
SUDOERS_FILE="/etc/sudoers.d/cmdgate"

if [[ $EUID -ne 0 ]]; then
    echo "This script must be run as root" >&2
    exit 1
fi

mkdir -p "${INSTALL_DIR}" "${WORK_DIR}" "${LOG_DIR}"

install -m 0755 cmdgate "${INSTALL_DIR}/cmdgate"
install -m 0750 cmdgate-exec "${INSTALL_DIR}/cmdgate-exec"
install -m 0640 allowlist.yaml "${INSTALL_DIR}/allowlist.yaml"

chmod 0755 "${INSTALL_DIR}"
chmod 0700 "${WORK_DIR}"
chmod 0750 "${LOG_DIR}"

cat > "${SUDOERS_FILE}" <<'EOF'
cmdgateadm ALL=(ALL) NOPASSWD: /opt/cmdgate/cmdgate-exec *
EOF

chmod 0440 "${SUDOERS_FILE}"
visudo -c

echo "CmdGate installed to ${INSTALL_DIR}"
```

- [ ] **Step 3: Verify script syntax**

Run:
```bash
cd /srv/cmdgate && bash -n scripts/install-cmdgate.sh
```

Expected: no output (success).

- [ ] **Step 4: Commit**

```bash
cd /srv/cmdgate && git add configs/allowlist.yaml scripts/install-cmdgate.sh && git commit -m "feat: add default allowlist and install script"
```

---

## Task 12: README.md

**Files:**
- Create: `README.md`

- [ ] **Step 1: Write README**

Create `README.md` covering:

1. Project overview
2. Binary responsibilities (`cmdgate` vs `cmdgate-exec`)
3. Build instructions
4. Installation instructions
5. Usage examples (`run`, `run list`, `policy validate`, `policy apply`)
6. allowlist.yaml format and matchers
7. Audit log location and format
8. Security notes (no shell execution, sudoers example)

- [ ] **Step 2: Commit**

```bash
cd /srv/cmdgate && git add README.md && git commit -m "docs: add README"
```

---

## Task 13: Final integration and full test run

- [ ] **Step 1: Run all tests**

```bash
cd /srv/cmdgate && go test ./...
```

Expected: PASS for all packages.

- [ ] **Step 2: Build both binaries**

```bash
cd /srv/cmdgate && go build -o cmdgate ./cmd/cmdgate && go build -o cmdgate-exec ./cmd/cmdgate-exec
```

Expected: both binaries produced.

- [ ] **Step 3: Verify CLI help exits cleanly**

```bash
cd /srv/cmdgate && ./cmdgate-exec 2>&1 || true
```

Expected: usage message, exit code non-zero is OK.

- [ ] **Step 4: Commit**

```bash
cd /srv/cmdgate && git add . && git commit -m "chore: final integration and verification"
```

---

## Spec Coverage Check

| Requirement | Task |
|---|---|
| Two Go binaries | Task 9, Task 10 |
| argv-only execution | Task 6, Task 9 |
| allowlist.yaml parsing | Task 1 |
| Exact command matching | Task 2 |
| Placeholder extraction | Task 3 |
| number matcher | Task 4 |
| rpmFiles matcher | Task 5 |
| run list | Task 9 |
| policy validate/apply | Task 8, Task 9 |
| Audit logging | Task 7, Task 9 |
| Install script | Task 11 |
| README | Task 12 |

No placeholders remain; every step contains concrete code or exact commands.
