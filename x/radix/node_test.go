package radix

import (
	"testing"
	"reflect"
)

func stringLabel(s string) Label {
	return StringLabel(s)
}

func globLabel(s string) Label {
	return NewGlobLabel(s)
}

func TestEdges_Less(t *testing.T) {
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

	for _, c := range cases {
		var e edges
		e = append(e, edge{label: c.fn(c.x)}, edge{label: c.fn(c.y)})
		if e.Less(0, 1) != c.ok {
			t.Fatal("bad case:", e)
		}
	}
}

func TestEdges_Sort(t *testing.T) {
	type exp struct {
		x, y []string
		fn   func(string) Label
	}

	cases := []exp{
		{
			[]string{"x", "*", "x*", "*x", "**"},
			[]string{"*", "**", "*x", "x", "x*"},
			stringLabel,
		},
		{
			[]string{"x", "*", "x*", "*x", "**"},
			[]string{"x", "*x", "x*", "*", "**"},
			globLabel,
		},
		{
			[]string{"x", "**", "x*", "*x", "*"},
			[]string{"x", "*x", "x*", "**", "*"},
			globLabel,
		},
		{
			[]string{"x", "**", "*x*", "*x", "*"},
			[]string{"x", "*x", "*x*", "**", "*"},
			globLabel,
		},
		{
			[]string{"x", "*x*", "**", "x*", "*x", "*"},
			[]string{"x", "*x", "x*", "*x*", "**", "*"},
			globLabel,
		},
	}

	for i, c := range cases {
		var e edges
		for _, z := range c.x {
			e = append(e, edge{label: c.fn(z)})
			e.Sort()
		}
		z := make([]string, 0, e.Len())
		for _, v := range e {
			z = append(z, v.label.String())
		}
		if !reflect.DeepEqual(z, c.y) {
			t.Fatalf("bad case %v: expected %v, got %v", i + 1, c.y, z)
		}
	}
}

func TestEdges_Search(t *testing.T) {
	type exp struct {
		x     []string
		y     string
		found []int
		fn    func(string) Label
	}

	cases := []exp{
		{
			[]string{"*", "**", "*x", "x", "x*"},
			"x",
			[]int{3},
			stringLabel,
		},
		{
			[]string{"x", "*x", "x*", "*", "**"},
			"x",
			[]int{0, 1, 2, 3, 4},
			globLabel,
		},
		{
			[]string{"x", "*x", "x*", "**", "*"},
			"xy",
			[]int{2, 3, 4},
			globLabel,
		},
		{
			[]string{"x", "*x", "*x*", "**", "*"},
			"y",
			[]int{3, 4},
			globLabel,
		},
		{
			[]string{"x", "*x", "x*", "*x*", "**", "*"},
			"yx",
			[]int{1, 3, 4, 5},
			globLabel,
		},
		{
			[]string{"xy", "x*", "yx", "*x*", "z", "*z"},
			"x",
			[]int{1, 3},
			globLabel,
		},
		{
			[]string{"x", "*x", "xy", "x*", "yx", "*x*", "z", "*z"},
			"x",
			[]int{0, 1, 3, 5},
			globLabel,
		},
		{
			[]string{"x", "*x", "xy", "x*", "yx", "*x*", "z", "*z"},
			"y",
			nil,
			globLabel,
		},
		{
			[]string{"x", "*x", "xy", "x*", "yx", "*x*", "z", "*z"},
			"xy",
			[]int{2, 3, 5},
			globLabel,
		},
		{
			[]string{"x", "*x", "xy", "x*", "yx", "*x*", "z", "*z"},
			"yx",
			[]int{4, 5},
			globLabel,
		},
	}

	for i, c := range cases {
		var e edges
		for _, z := range c.x {
			e = append(e, edge{label: c.fn(z)})
			e.Sort()
		}
		z := make([]string, 0, e.Len())
		for _, v := range e {
			z = append(z, v.label.String())
		}
		if !reflect.DeepEqual(z, c.x) {
			t.Fatalf("bad case %v: expected %v, got %v", i + 1, c.x, z)
		}
		found := e.Search(c.fn(c.y))
		if !reflect.DeepEqual(found, c.found) {
			t.Fatalf("bad case %v, expected %v, got %v", i + 1, c.found, found)
		}
	}
}
