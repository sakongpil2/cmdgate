package matchers

import (
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// RpmFilesMatcher validates a list of RPM files by querying their NAME field.
type RpmFilesMatcher struct {
	// MetadataNameIn is the allowlist of RPM NAME values permitted by this matcher.
	MetadataNameIn []string
	// Multiple allows more than one RPM path to be supplied at once.
	Multiple bool
	// AllowedDirs restricts RPM paths to the given directory prefixes.
	// When empty, only the absolute-path check is enforced.
	AllowedDirs []string
	// RpmQuery returns the query string for a given RPM file path.
	// When nil, defaultRpmQuery is used.
	RpmQuery func(path string) (string, error)
}

// Validate ensures every RPM file path is absolute and its NAME is allowed.
func (r *RpmFilesMatcher) Validate(paths []string) error {
	if !r.Multiple && len(paths) > 1 {
		return fmt.Errorf("multiple rpm files not allowed")
	}
	if r.RpmQuery == nil {
		r.RpmQuery = defaultRpmQuery
	}
	for _, p := range paths {
		if !filepath.IsAbs(p) {
			return fmt.Errorf("rpm path must be absolute: %q", p)
		}
		clean := filepath.Clean(p)
		if len(r.AllowedDirs) > 0 && !hasAllowedPrefix(clean, r.AllowedDirs) {
			return fmt.Errorf("rpm path %q is not under allowed directories", p)
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

func hasAllowedPrefix(path string, dirs []string) bool {
	for _, d := range dirs {
		dir := filepath.Clean(d)
		if path == dir || strings.HasPrefix(path, dir+string(filepath.Separator)) {
			return true
		}
	}
	return false
}

func (r *RpmFilesMatcher) allowed(name string) bool {
	for _, n := range r.MetadataNameIn {
		if n == name {
			return true
		}
	}
	return false
}

func defaultRpmQuery(path string) (string, error) {
	out, err := exec.Command("rpm", "-qp", "--queryformat", "%{NAME} %{VERSION} %{RELEASE} %{ARCH}\\n", path).Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return "", fmt.Errorf("%w: %s", err, exitErr.Stderr)
		}
		return "", err
	}
	return string(out), nil
}
