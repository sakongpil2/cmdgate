package matchers

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRpmFilesMatcherAllAllowed(t *testing.T) {
	dir := t.TempDir()
	createFakeRPM(t, filepath.Join(dir, "kubelet-1.rpm"), "kubelet")
	createFakeRPM(t, filepath.Join(dir, "kubeadm-1.rpm"), "kubeadm")

	m := RpmFilesMatcher{
		MetadataNameIn: []string{"kubelet", "kubeadm"},
		Multiple:       true,
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

func TestRpmFilesMatcherRelativePath(t *testing.T) {
	m := RpmFilesMatcher{
		MetadataNameIn: []string{"kubelet"},
		RpmQuery:       fakeRpmQuery,
	}
	if err := m.Validate([]string{"relative/kubelet-1.rpm"}); err == nil {
		t.Error("expected relative path to be rejected")
	}
}

func TestRpmFilesMatcherDefaultsToRealRpmQuery(t *testing.T) {
	m := RpmFilesMatcher{MetadataNameIn: []string{"kubelet"}}
	if err := m.Validate([]string{"/nonexistent/kubelet-1.rpm"}); err == nil {
		t.Error("expected default rpm query to run and fail on nonexistent file")
	}
}

func TestRpmFilesMatcherEmptyAllowlist(t *testing.T) {
	dir := t.TempDir()
	createFakeRPM(t, filepath.Join(dir, "kubelet-1.rpm"), "kubelet")

	m := RpmFilesMatcher{
		MetadataNameIn: []string{},
		RpmQuery:       fakeRpmQuery,
	}
	if err := m.Validate([]string{filepath.Join(dir, "kubelet-1.rpm")}); err == nil {
		t.Error("expected empty allowlist to reject all names")
	}
}

func TestRpmFilesMatcherQueryError(t *testing.T) {
	dir := t.TempDir()
	createFakeRPM(t, filepath.Join(dir, "kubelet-1.rpm"), "kubelet")

	m := RpmFilesMatcher{
		MetadataNameIn: []string{"kubelet"},
		RpmQuery:       func(string) (string, error) { return "", errors.New("query error") },
	}
	if err := m.Validate([]string{filepath.Join(dir, "kubelet-1.rpm")}); err == nil {
		t.Error("expected query error to be rejected")
	}
}

func TestRpmFilesMatcherEmptyQueryOutput(t *testing.T) {
	dir := t.TempDir()
	createFakeRPM(t, filepath.Join(dir, "kubelet-1.rpm"), "kubelet")

	m := RpmFilesMatcher{
		MetadataNameIn: []string{"kubelet"},
		RpmQuery:       func(string) (string, error) { return "", nil },
	}
	if err := m.Validate([]string{filepath.Join(dir, "kubelet-1.rpm")}); err == nil {
		t.Error("expected empty query output to be rejected")
	}
}

func TestRpmFilesMatcherRejectsMultipleWhenNotAllowed(t *testing.T) {
	dir := t.TempDir()
	createFakeRPM(t, filepath.Join(dir, "kubelet-1.rpm"), "kubelet")
	createFakeRPM(t, filepath.Join(dir, "kubeadm-1.rpm"), "kubeadm")

	m := RpmFilesMatcher{
		MetadataNameIn: []string{"kubelet", "kubeadm"},
		Multiple:       false,
		RpmQuery:       fakeRpmQuery,
	}
	if err := m.Validate([]string{
		filepath.Join(dir, "kubelet-1.rpm"),
		filepath.Join(dir, "kubeadm-1.rpm"),
	}); err == nil {
		t.Error("expected multiple paths to be rejected when Multiple=false")
	}
}

func TestRpmFilesMatcherAllowedDirs(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "rpms")
	nearby := filepath.Join(dir, "rpms-other")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(nearby, 0o755); err != nil {
		t.Fatal(err)
	}
	createFakeRPM(t, filepath.Join(sub, "kubelet-1.rpm"), "kubelet")
	createFakeRPM(t, filepath.Join(nearby, "kubelet-1.rpm"), "kubelet")

	m := RpmFilesMatcher{
		MetadataNameIn: []string{"kubelet"},
		AllowedDirs:    []string{sub},
		RpmQuery:       fakeRpmQuery,
	}
	if err := m.Validate([]string{filepath.Join(sub, "kubelet-1.rpm")}); err != nil {
		t.Errorf("expected allowed dir path to be accepted, got %v", err)
	}

	m.AllowedDirs = []string{"/other"}
	if err := m.Validate([]string{filepath.Join(sub, "kubelet-1.rpm")}); err == nil {
		t.Error("expected path outside allowed dirs to be rejected")
	}

	m.AllowedDirs = []string{sub}
	if err := m.Validate([]string{filepath.Join(nearby, "kubelet-1.rpm")}); err == nil {
		t.Error("expected path with only a shared string prefix to be rejected")
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
	case strings.Contains(name, "kubelet"):
		return "kubelet 1.0 1 x86_64", nil
	case strings.Contains(name, "kubeadm"):
		return "kubeadm 1.0 1 x86_64", nil
	default:
		return "sshd 1.0 1 x86_64", nil
	}
}
