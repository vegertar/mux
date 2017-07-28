package radix

import (
	"testing"
	"reflect"
)

func TestGlobLabel_Match(t *testing.T) {
	type exp struct {
		pattern, s string
		matched    bool
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
			t.Errorf("bad case: %v", c)
		}
	}
}

func TestNewCharKey(t *testing.T) {
	type exp struct {
		x string
		y []string
	}

	cases := []exp{
		{"", nil},
		{"xyz", []string{"x", "y", "z"}},
		{"你好, China", []string{"你", "好", ",", " ", "C", "h", "i", "n", "a"}},
	}
	for i, c := range cases {
		y := NewCharKey(c.x).Strings()
		if !reflect.DeepEqual(y, c.y) {
			t.Errorf("bad case %v: expected %v, got %v", i + 1, c.y, y)
		}
	}
}
