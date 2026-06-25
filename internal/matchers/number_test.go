package matchers

import "testing"

func TestNumberMatcher_Validate(t *testing.T) {
	m := NumberMatcher{}

	cases := []struct {
		value string
		valid bool
	}{
		{"123", true},
		{"0", true},
		{"-1", true},
		{"abc", false},
		{"", false},
		{"1.5", false},
		{" 123", false},
	}

	for _, tc := range cases {
		err := m.Validate(tc.value)
		if tc.valid && err != nil {
			t.Errorf("Validate(%q): expected valid, got error: %v", tc.value, err)
		}
		if !tc.valid && err == nil {
			t.Errorf("Validate(%q): expected invalid, got no error", tc.value)
		}
	}
}
