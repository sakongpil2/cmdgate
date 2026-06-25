package matchers

import "testing"

func TestStringMatcher_Validate(t *testing.T) {
	m := StringMatcher{}
	if err := m.Validate("hello"); err != nil {
		t.Errorf("expected valid, got %v", err)
	}
	if err := m.Validate(""); err == nil {
		t.Error("expected empty value to be rejected")
	}
}

func TestStringMatcher_ValidatePattern(t *testing.T) {
	m := StringMatcher{Pattern: "^[a-z0-9-]+$"}
	if err := m.Validate("kubelet"); err != nil {
		t.Errorf("expected valid, got %v", err)
	}
	if err := m.Validate("Kubelet"); err == nil {
		t.Error("expected uppercase to be rejected by pattern")
	}
}

func TestStringMatcher_InvalidPattern(t *testing.T) {
	m := StringMatcher{Pattern: "[invalid"}
	if err := m.Validate("hello"); err == nil {
		t.Error("expected invalid pattern to be rejected")
	}
}
