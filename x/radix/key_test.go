package radix

import "testing"

func TestGlobLabel_Match(t *testing.T) {
	type exp struct {
		pattern, s string
		matched bool
	}

	cases := []exp{
		{"", "", true},
		{"*", "ab", true},
		{"ab", "ab", true},
		{"a*b", "ab", true},
		{"*ab", "ab", true},
		{"ab*", "ab", true},
		{"ab*", "a123b", false},
		{"a*b", "a123b", true},
		{"a**", "a123b", true},
		{"*ab", "a123b", false},
		{"**b", "a123b", true},
		{"a*b", "a123bc", false},
		{"a*b*", "a123bc", true},
		{"a*b*", "0a123bc", false},
		{"*a*b", "0a123b", true},
		{"*a*b", "0a123bc", false},
		{"*a*b*", "0a123bc", true},
	}
	for _, c := range cases {
		label := NewGlobLabel(c.pattern)
		if label.Match(c.s) != c.matched {
			t.Fatalf("bad case: %v", c)
		}
	}
}
