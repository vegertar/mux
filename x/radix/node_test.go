package radix

import (
	"testing"
	"sort"
	"reflect"
)

func stringLabel(s string) Label {
	return StringLabel(s)
}

func globLabel(s string) Label {
	return NewGlobLabel(s)
}

func TestLessLabel(t *testing.T) {
	type exp struct {
		x, y string
		ok   bool
		fn   func(string) Label
	}

	cases := []exp{
		{"x", "x", false, stringLabel},
		{"x", "x", false, globLabel},
		{"x", "y", true, stringLabel},
		{"x", "y", true, globLabel},
		{"y", "x", false, stringLabel},
		{"y", "x", false, globLabel},
		{"x", "*", false, stringLabel},
		{"x", "*", true, globLabel},
		{"*", "x", true, stringLabel},
		{"*", "x", false, globLabel},
		{"*", "*", false, stringLabel},
		{"*", "*", false, globLabel},
		{"x", "x*", true, stringLabel},
		{"x", "x*", true, globLabel},
		{"x", "*x", false, stringLabel},
		{"x", "*x", true, globLabel},
		{"x*", "x", false, stringLabel},
		{"x*", "x", false, globLabel},
	}

	for i, c := range cases {
		x, y := c.fn(c.x), c.fn(c.y)
		if lessLabel(x, y) != c.ok {
			t.Errorf("bad case %v: %v", i + 1, c)
		}
	}
}

func TestLessKey(t *testing.T) {
	type exp struct {
		x, y []string
		ok   bool
		fn   func([]string) Key
	}

	cases := []exp{
		{[]string{"x"}, []string{"x"}, false, NewStringSliceKey},
		{[]string{"x"}, []string{"x"}, false, NewGlobSliceKey},
		{[]string{"x"}, []string{"y"}, true, NewStringSliceKey},
		{[]string{"x"}, []string{"y"}, true, NewGlobSliceKey},
		{[]string{"y"}, []string{"x"}, false, NewStringSliceKey},
		{[]string{"y"}, []string{"x"}, false, NewGlobSliceKey},
		{[]string{"x"}, []string{"*"}, false, NewStringSliceKey},
		{[]string{"x"}, []string{"*"}, true, NewGlobSliceKey},
		{[]string{"*"}, []string{"x"}, true, NewStringSliceKey},
		{[]string{"*"}, []string{"x"}, false, NewGlobSliceKey},
		{[]string{"*"}, []string{"*"}, false, NewStringSliceKey},
		{[]string{"*"}, []string{"*"}, false, NewGlobSliceKey},
		{[]string{"x"}, []string{"x*"}, true, NewStringSliceKey},
		{[]string{"x"}, []string{"x*"}, true, NewGlobSliceKey},
		{[]string{"x"}, []string{"*x"}, false, NewStringSliceKey},
		{[]string{"x"}, []string{"*x"}, true, NewGlobSliceKey},
		{[]string{"x*"}, []string{"x"}, false, NewStringSliceKey},
		{[]string{"x*"}, []string{"x"}, false, NewGlobSliceKey},
	}

	for i, c := range cases {
		x, y := c.fn(c.x), c.fn(c.y)
		if lessKey(x, y) != c.ok {
			t.Errorf("bad case %v: %v", i + 1, c)
		}
	}
}

func TestNode_Search(t *testing.T) {
	input := []struct {
		x  string
		fn func(string) Label
	}{
		{"/v1/x", stringLabel},
		{"/v2/x", stringLabel},
		{"/v*/x", globLabel},
		{"/v3/*", globLabel},
	}

	n := new(node)
	for _, c := range input {
		n.addEdge(edge{
			label: c.fn(c.x),
		})
	}

	output := []struct{
		x string
		y []string
	}{
		{"/v1", nil},
		{"/v3", nil},
		{"/v3/", []string{"/v3/*"}},
		{"/v3/x", []string{"/v*/x", "/v3/*"}},
		{"/v1/x", []string{"/v1/x", "/v*/x"}},
		{"/v2/x", []string{"/v2/x", "/v*/x"}},
	}

	for i, c := range output {
		edges := n.search(stringLabel(c.x))
		sort.Sort(sortEdgeByPattern(edges))

		var l []Label
		for _, e := range edges {
			l = append(l, e.label)
		}

		y := Key(l).Strings()
		if !reflect.DeepEqual(y, c.y) {
			t.Errorf("bad case %v: expected %v, got %v", i + 1, c.y, y)
		}
	}
}
