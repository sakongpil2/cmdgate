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

	"github.com/sakongpil2/cmdgate/internal/allowlist"
	"gopkg.in/yaml.v3"
)

// Manifest describes the policy bundle metadata.
type Manifest struct {
	Version   string `yaml:"version"`
	Timestamp string `yaml:"timestamp"`
}

// ValidateBundle checks that a policy bundle is well-formed and consistent.
func ValidateBundle(bundlePath string) error {
	_, _, err := extractAndVerify(bundlePath)
	return err
}

func extractAndVerify(bundlePath string) (allowlistBytes []byte, manifest Manifest, err error) {
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
			allowlistBytes, err = io.ReadAll(tr)
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
	sum := fmt.Sprintf("%x", sha256.Sum256(allowlistBytes))
	if sum != checksum {
		return nil, manifest, fmt.Errorf("checksum mismatch")
	}
	cfg, err := allowlist.Parse(allowlistBytes)
	if err != nil {
		return nil, manifest, fmt.Errorf("invalid allowlist.yaml: %w", err)
	}
	if err := cfg.ValidateSchema(); err != nil {
		return nil, manifest, fmt.Errorf("invalid allowlist schema: %w", err)
	}
	return allowlistBytes, manifest, nil
}
