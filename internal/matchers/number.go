package matchers

import (
	"fmt"
	"strconv"
)

// NumberMatcher validates that a value is a base-10 integer.
type NumberMatcher struct{}

// Validate reports whether value is a valid base-10 integer.
func (n NumberMatcher) Validate(value string) error {
	if _, err := strconv.Atoi(value); err != nil {
		return fmt.Errorf("%q is not a valid number", value)
	}
	return nil
}
