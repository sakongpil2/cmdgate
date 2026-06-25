package matchers

import "testing"

func TestStringMatcherPatternRejectsDotDot(t *testing.T) {
	m := StringMatcher{Pattern: `^(?:[a-zA-Z0-9_-]+/)*[a-zA-Z0-9_-]+\.sh$`}
	cases := []struct {
		value string
		want  bool
	}{
		{"backup.sh", true},
		{"maintenance/reboot.sh", true},
		{"../../etc/passwd.sh", false},
		{"/etc/passwd.sh", false},
		{"script", false},
	}
	for _, tc := range cases {
		err := m.Validate(tc.value)
		got := err == nil
		if got != tc.want {
			t.Errorf("Validate(%q) error = %v, want valid=%v", tc.value, err, tc.want)
		}
	}
}
