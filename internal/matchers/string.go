package matchers

import (
	"fmt"
	"regexp"
)

// StringMatcher validates that a placeholder value is a non-empty string.
// An optional regular expression Pattern can restrict allowed values.
type StringMatcher struct {
	Pattern string
}

// Validate returns an error if value is empty or does not match Pattern.
func (s StringMatcher) Validate(value string) error {
	if value == "" {
		return fmt.Errorf("value is empty")
	}
	if s.Pattern == "" {
		return nil
	}
	re, err := regexp.Compile(s.Pattern)
	if err != nil {
		return fmt.Errorf("invalid pattern %q: %w", s.Pattern, err)
	}
	if !re.MatchString(value) {
		return fmt.Errorf("%q does not match pattern %q", value, s.Pattern)
	}
	return nil
}
