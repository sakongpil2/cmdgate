package policy

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestValidateBundle(t *testing.T) {
	dir := t.TempDir()
	bundle := filepath.Join(dir, "bundle.tar.gz")
	createBundle(t, bundle, "1.1.0", false)

	if err := ValidateBundle(bundle); err != nil {
		t.Errorf("expected valid bundle, got %v", err)
	}
}

func TestValidateBundle_missingFile(t *testing.T) {
	dir := t.TempDir()
	bundle := filepath.Join(dir, "bundle.tar.gz")
	createBundle(t, bundle, "1.1.0", true)

	if err := ValidateBundle(bundle); err == nil {
		t.Error("expected error for bundle missing required file, got nil")
	}
}

func TestValidateBundle_checksumMismatch(t *testing.T) {
	dir := t.TempDir()
	bundle := filepath.Join(dir, "bundle.tar.gz")
	createBundle(t, bundle, "1.1.0", false)

	if err := overwriteTarFile(bundle, "checksums.sha256", "deadbeef"); err != nil {
		t.Fatalf("overwrite checksum: %v", err)
	}

	if err := ValidateBundle(bundle); err == nil {
		t.Error("expected checksum mismatch error, got nil")
	}
}

func TestValidateBundle_missingVersion(t *testing.T) {
	dir := t.TempDir()
	bundle := filepath.Join(dir, "bundle.tar.gz")
	createBundle(t, bundle, "", false)

	if err := ValidateBundle(bundle); err == nil {
		t.Error("expected error for missing manifest version, got nil")
	}
}

func TestValidateBundle_invalidAllowlistYAML(t *testing.T) {
	dir := t.TempDir()
	bundle := filepath.Join(dir, "bundle.tar.gz")
	createBundle(t, bundle, "1.1.0", false)

	invalidBody := "[not: valid: yaml:"
	if err := overwriteTarFile(bundle, "allowlist.yaml", invalidBody); err != nil {
		t.Fatalf("overwrite allowlist: %v", err)
	}
	checksum := fmt.Sprintf("%x", sha256.Sum256([]byte(invalidBody)))
	if err := overwriteTarFile(bundle, "checksums.sha256", checksum); err != nil {
		t.Fatalf("overwrite checksum: %v", err)
	}

	if err := ValidateBundle(bundle); err == nil {
		t.Error("expected error for invalid allowlist YAML, got nil")
	}
}

func createBundle(t *testing.T, path, version string, omitChecksum bool) {
	t.Helper()
	allowlistBody := "version: " + version + "\nmode: allowlist-only\ncommands: []\n"
	checksum := fmt.Sprintf("%x", sha256.Sum256([]byte(allowlistBody)))

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
		"manifest.yaml":  "version: " + version + "\ntimestamp: 2026-06-25T00:00:00Z\n",
		"allowlist.yaml": allowlistBody,
	}
	if !omitChecksum {
		files["checksums.sha256"] = checksum
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

func overwriteTarFile(bundlePath, name, body string) error {
	f, err := os.Open(bundlePath)
	if err != nil {
		return err
	}
	defer f.Close()

	gr, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gr.Close()

	entries := make(map[string]string)
	tr := tar.NewReader(gr)
	for {
		h, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		b, err := io.ReadAll(tr)
		if err != nil {
			return err
		}
		entries[h.Name] = string(b)
	}
	entries[name] = body

	out, err := os.Create(bundlePath)
	if err != nil {
		return err
	}
	defer out.Close()
	gw := gzip.NewWriter(out)
	defer gw.Close()
	tw := tar.NewWriter(gw)
	defer tw.Close()
	for n, b := range entries {
		h := &tar.Header{Name: n, Mode: 0o644, Size: int64(len(b))}
		if err := tw.WriteHeader(h); err != nil {
			return err
		}
		if _, err := tw.Write([]byte(b)); err != nil {
			return err
		}
	}
	return nil
}
