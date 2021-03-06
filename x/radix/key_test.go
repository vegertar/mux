package radix

import (
	"reflect"
	"testing"
)

func TestGlobLabel_Match(t *testing.T) {
	type exp struct {
		pattern, s string
		matched    bool
	}

	cases := []exp{
		{"", "", true},
		{"*", "", true},
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
		label := newGlobLabel(c.pattern)
		if label.Match(c.s) != c.matched {
			t.Errorf("bad case: %v", c)
		}
	}
}

func TestGlobKey_Match(t *testing.T) {
	type exp struct {
		pattern, s []string
		matched    bool
	}

	cases := []exp{
		{[]string{""}, []string{""}, true},
		{[]string{"*"}, []string{""}, true},
		{[]string{"*"}, []string{"ab"}, true},
		{[]string{"*", "baz"}, []string{"1", "x"}, false},
		{[]string{"baz", "*"}, []string{"1", "x"}, false},
		{[]string{"a", "b"}, []string{"a", "b"}, true},
		{[]string{"**"}, []string{"a", "b"}, true},
		{[]string{"a", "**", "b"}, []string{"a", "b"}, true},
		{[]string{"**", "a", "b"}, []string{"a", "b"}, true},
		{[]string{"a", "b", "**"}, []string{"a", "b"}, true},
		{[]string{"a", "b", "**"}, []string{"a", "1", "2", "3", "b"}, false},
		{[]string{"a", "**", "b"}, []string{"a", "1", "2", "3", "b"}, true},
		{[]string{"a", "**", "**"}, []string{"a", "1", "2", "3", "b"}, true},
		{[]string{"**", "**", "a", "b"}, []string{"a", "1", "2", "3", "b"}, false},
		{[]string{"**", "**", "b"}, []string{"a", "1", "2", "3", "b"}, true},
		{[]string{"a", "**", "b"}, []string{"a", "1", "2", "3", "b", "c"}, false},
		{[]string{"a", "**", "b", "**"}, []string{"a", "1", "2", "3", "b", "c"}, true},
		{[]string{"a", "**", "b", "**"}, []string{"0", "a", "1", "2", "3", "b", "c"}, false},
		{[]string{"**", "a", "**", "b"}, []string{"0", "a", "1", "2", "3", "b"}, true},
		{[]string{"**", "a", "**", "b"}, []string{"0", "a", "1", "2", "3", "b", "c"}, false},
		{[]string{"**", "a", "**", "b", "**"}, []string{"0", "a", "1", "2", "3", "b", "c"}, true},
	}
	for i, c := range cases {
		key := NewGlobSliceKey(c.pattern)
		if key.Match(NewGlobSliceKey(c.s)) != c.matched {
			t.Errorf("bad case %d: %v", i+1, c)
		}
	}
}

func TestGlobKey_Capture(t *testing.T) {
	type exp struct {
		pattern, x []string
		y          [][]string
	}

	cases := []exp{
		{[]string{""}, []string{""}, nil},
		{[]string{"*"}, []string{""}, [][]string{[]string{""}}},
		{[]string{"*"}, []string{"ab"}, [][]string{[]string{"ab"}}},
		{[]string{"a", "b"}, []string{"a", "b"}, nil},
		{[]string{"**"}, []string{"a", "b"}, [][]string{[]string{"a", "b"}}},
		{[]string{"a", "**", "b"}, []string{"a", "b"}, [][]string{nil}},
		{[]string{"a", "**", "**", "b"}, []string{"a", "b"}, [][]string{nil, nil}},
		{[]string{"**", "a", "b"}, []string{"a", "b"}, [][]string{nil}},
		{[]string{"a", "b", "**"}, []string{"a", "b"}, [][]string{nil}},
		{[]string{"a", "**", "b"}, []string{"a", "1", "2", "3", "b"}, [][]string{[]string{"1", "2", "3"}}},
		{[]string{"a", "**", "**", "b"}, []string{"a", "1", "2", "3", "b"}, [][]string{[]string{"1", "2", "3"}, nil}},
		{[]string{"a", "**", "**"}, []string{"a", "1", "2", "3", "b"}, [][]string{[]string{"1", "2", "3", "b"}, nil}},
		{[]string{"**", "**", "b"}, []string{"a", "1", "2", "3", "b"}, [][]string{[]string{"a", "1", "2", "3"}, nil}},
		{[]string{"a", "**", "b", "**"}, []string{"a", "1", "2", "3", "b", "c"}, [][]string{[]string{"1", "2", "3"}, []string{"c"}}},
		{[]string{"**", "a", "**", "b"}, []string{"0", "a", "1", "2", "3", "b"}, [][]string{[]string{"0"}, []string{"1", "2", "3"}}},
		{[]string{"**", "a", "**", "b", "**"}, []string{"0", "a", "1", "2", "3", "b", "c"}, [][]string{[]string{"0"}, []string{"1", "2", "3"}, []string{"c"}}},
	}
	for i, c := range cases {
		key := NewGlobSliceKey(c.pattern)
		var y [][]string
		for _, k := range key.Capture(NewGlobSliceKey(c.x)) {
			y = append(y, k.Strings())
		}
		if !reflect.DeepEqual(y, c.y) {
			t.Errorf("bad case %d, expected %v, got %v", i+1, c.y, y)
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
			t.Errorf("bad case %v: expected %v, got %v", i+1, c.y, y)
		}
	}
}
