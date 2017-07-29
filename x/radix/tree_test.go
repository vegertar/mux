package radix

import (
	"math/rand"
	"reflect"
	"sort"
	"testing"
)

func randomString(n int) string {
	b := make([]byte, 0, n)
	for i := 0; i < n; i++ {
		b = append(b, byte(rand.Intn(95)+32))
	}
	return string(b)
}

func randomStringSlice(size, unit int) []string {
	values := make([]string, 0, size)
	for i := 0; i < size; i++ {
		values = append(values, randomString(unit))
	}
	return values
}

func TestTree(t *testing.T) {
	input := randomStringSlice(1000, 20)
	sort.Strings(input)

	r := New()
	for i := 0; i < len(input); i++ {
		s := input[i]
		r.Insert(NewCharKey(s), i)
	}

	if r.Len() != len(input) {
		t.Fatalf("bad tree length: expected %d, got %d", len(input), r.Len())
	}

	// Check walking in order
	r.Walk(func(leaf Leaf) bool {
		k, v := leaf.Key, leaf.Value
		i := v.(int)
		s := input[i]

		if s != k.StringWith("") {
			t.Fatalf("bad key: expected %v, got %v", s, k)
		}
		return false
	})

	for i, k := range input {
		out := r.Get(NewCharKey(k))
		if len(out) != 1 {
			t.Fatalf("missing key: %v", k)
		}
		if out[0].Value != i {
			t.Fatalf("value mis-match: %v %v", out, i)
		}
	}

	for i, k := range input {
		out, ok := r.Delete(NewCharKey(k))
		if !ok {
			t.Fatalf("missing key: %v", k)
		}
		if out != i {
			t.Fatalf("value mis-match: %v %v", out, i)
		}
	}
	if r.Len() != 0 {
		t.Fatalf("bad length: %v", r.Len())
	}
}

func TestRoot(t *testing.T) {
	r := New()
	k := NewCharKey("")
	_, ok := r.Delete(k)
	if ok {
		t.Fatalf("bad")
	}
	_, ok = r.Insert(k, true)
	if ok {
		t.Fatalf("bad")
	}
	leaves := r.Get(k)
	if len(leaves) != 1 || leaves[0].Value != true {
		t.Fatalf("bad: %v", leaves)
	}
	val, ok := r.Delete(k)
	if !ok || val != true {
		t.Fatalf("bad: %v", val)
	}
}

func TestDelete(t *testing.T) {
	r := New()
	keys := []string{"", "A", "AB"}

	for _, key := range keys {
		r.Insert(NewCharKey(key), true)
	}

	for _, key := range keys {
		_, ok := r.Delete(NewCharKey(key))
		if !ok {
			t.Fatalf("bad %v", key)
		}
	}
}

func TestDeletePrefix(t *testing.T) {
	type exp struct {
		x          []string
		prefix     string
		y          []string
		numDeleted int
	}

	cases := []exp{
		{[]string{"", "A", "AB", "ABC", "R", "S"}, "A", []string{"", "R", "S"}, 3},
		{[]string{"", "A", "AB", "ABC", "R", "S"}, "ABC", []string{"", "A", "AB", "R", "S"}, 1},
		{[]string{"", "A", "AB", "ABC", "R", "S"}, "", nil, 6},
		{[]string{"", "A", "AB", "ABC", "R", "S"}, "S", []string{"", "A", "AB", "ABC", "R"}, 1},
		{[]string{"", "A", "AB", "ABC", "R", "S"}, "SS", []string{"", "A", "AB", "ABC", "R", "S"}, 0},
	}

	for i, c := range cases {
		r := New()
		for _, s := range c.x {
			r.Insert(NewCharKey(s), true)
		}

		deleted := r.DeletePrefix(NewCharKey(c.prefix))
		if deleted != c.numDeleted {
			t.Fatalf("Bad delete, expected %v to be deleted but got %v", c.numDeleted, deleted)
		}

		var y []string
		fn := func(leaf Leaf) bool {
			y = append(y, leaf.Key.StringWith(""))
			return false
		}
		r.Walk(fn)

		if !reflect.DeepEqual(y, c.y) {
			t.Fatalf("bad case %v: expected %v, got %v", i+1, c.y, y)
		}
	}
}

func TestLongestPrefix_Char(t *testing.T) {
	r := New()

	keys := []string{
		"",
		"foo",
		"foobar",
		"foobarbaz",
		"foobarbazzip",
		"foozip",
	}
	for _, k := range keys {
		r.Insert(NewCharKey(k), nil)
	}
	if r.Len() != len(keys) {
		t.Fatalf("bad len: %v %v", r.Len(), len(keys))
	}

	type exp struct {
		x string
		y string
	}
	cases := []exp{
		{"a", ""},
		{"abc", ""},
		{"fo", ""},
		{"foo", "foo"},
		{"foob", "foo"},
		{"foobar", "foobar"},
		{"foobarba", "foobar"},
		{"foobarbaz", "foobarbaz"},
		{"foobarbazzi", "foobarbaz"},
		{"foobarbazzip", "foobarbazzip"},
		{"foozi", "foo"},
		{"foozip", "foozip"},
		{"foozipzap", "foozip"},
	}
	for _, test := range cases {
		x := NewCharKey(test.x)
		leaves := r.LongestPrefix(x)
		if len(leaves) != 1 {
			var s []string
			for _, l := range leaves {
				s = append(s, l.Key.StringWith(""))
			}
			t.Fatalf("no match for %v, got %v", test, s)
		}
		k := leaves[0].Key
		if k.StringWith("") != test.y {
			t.Fatalf("mis-match: %v %v", k, test)
		}
	}
}

func TestLongestPrefix_Glob(t *testing.T) {
	r := New()

	keys := []string{
		"",
		"foo",
		"foo/bar",
		"foo/*/baz",
		"foo/bar/baz/zip",
		"foo/bar/baz/*",
		"foo/zip*",
	}
	for _, k := range keys {
		x := NewGlobKey(k, "/")
		old, updated := r.Insert(x, nil)
		if updated {
			t.Fatalf("bad updating: %v, %v", old, x)
		}
	}
	if r.Len() != len(keys) {
		t.Fatalf("bad len: %v %v", r.Len(), len(keys))
	}

	type exp struct {
		x string
		y []string
	}
	cases := []exp{
		{"/a", []string{""}},
		{"/abc", []string{""}},
		{"/fo", []string{""}},
		{"foo", []string{"foo"}},
		{"foo/b", []string{"foo"}},
		{"foo/bar", []string{"foo/bar"}},
		{"foo/bar/ba", []string{"foo/bar"}},
		{"foo/bar/baz", []string{"foo/*/baz"}},
		{"foo/bar/baz/zi", []string{"foo/bar/baz/*"}},
		{"foo/bar/baz/zip", []string{"foo/bar/baz/zip", "foo/bar/baz/*"}},
		{"foo/zi", []string{"foo"}},
		{"foo/zip", []string{"foo/zip*"}},
		{"foo/zip1", []string{"foo/zip*"}},
	}
	for i, c := range cases {
		x := NewGlobKey(c.x, "/")
		var y []string
		for _, leaf := range r.LongestPrefix(x) {
			y = append(y, leaf.Key.StringWith("/"))
		}
		if !reflect.DeepEqual(y, c.y) {
			t.Errorf("bad case %v: expected %v, got %v", i+1, c.y, y)
		}
	}
}

func TestWalkPrefix(t *testing.T) {
	r := New()

	keys := []string{
		"foobar",
		"foo/bar/baz",
		"foo/baz/bar",
		"foo/zip/zap",
		"zipzap",
	}
	for _, k := range keys {
		r.Insert(NewCharKey(k), nil)
	}
	if r.Len() != len(keys) {
		t.Fatalf("bad len: %v %v", r.Len(), len(keys))
	}

	type exp struct {
		x string
		y []string
	}
	cases := []exp{
		{
			"f",
			[]string{"foobar", "foo/bar/baz", "foo/baz/bar", "foo/zip/zap"},
		},
		{
			"foo",
			[]string{"foobar", "foo/bar/baz", "foo/baz/bar", "foo/zip/zap"},
		},
		{
			"foob",
			[]string{"foobar"},
		},
		{
			"foo/",
			[]string{"foo/bar/baz", "foo/baz/bar", "foo/zip/zap"},
		},
		{
			"foo/b",
			[]string{"foo/bar/baz", "foo/baz/bar"},
		},
		{
			"foo/ba",
			[]string{"foo/bar/baz", "foo/baz/bar"},
		},
		{
			"foo/bar",
			[]string{"foo/bar/baz"},
		},
		{
			"foo/bar/baz",
			[]string{"foo/bar/baz"},
		},
		{
			"foo/bar/bazoo",
			[]string{},
		},
		{
			"z",
			[]string{"zipzap"},
		},
	}

	for _, test := range cases {
		y := []string{}
		fn := func(leaf Leaf) bool {
			y = append(y, leaf.Key.StringWith(""))
			return false
		}
		r.WalkPrefix(NewCharKey(test.x), fn)
		sort.Strings(y)
		sort.Strings(test.y)
		if !reflect.DeepEqual(y, test.y) {
			t.Fatalf("mis-match: %v %v", y, test.y)
		}
	}
}

func TestWalkPath(t *testing.T) {
	r := New()

	keys := []string{
		"foo",
		"foo/bar",
		"foo/bar/baz",
		"foo/baz/bar",
		"foo/zip/zap",
		"zipzap",
	}
	for _, k := range keys {
		r.Insert(NewCharKey(k), nil)
	}
	if r.Len() != len(keys) {
		t.Fatalf("bad len: %v %v", r.Len(), len(keys))
	}

	type exp struct {
		x string
		y []string
	}
	cases := []exp{
		{
			"f",
			[]string{},
		},
		{
			"foo",
			[]string{"foo"},
		},
		{
			"foo/",
			[]string{"foo"},
		},
		{
			"foo/ba",
			[]string{"foo"},
		},
		{
			"foo/bar",
			[]string{"foo", "foo/bar"},
		},
		{
			"foo/bar/baz",
			[]string{"foo", "foo/bar", "foo/bar/baz"},
		},
		{
			"foo/bar/bazoo",
			[]string{"foo", "foo/bar", "foo/bar/baz"},
		},
		{
			"z",
			[]string{},
		},
	}

	for _, test := range cases {
		y := []string{}
		fn := func(leaf Leaf) bool {
			y = append(y, leaf.Key.StringWith(""))
			return false
		}
		r.WalkPath(NewCharKey(test.x), fn)
		sort.Strings(y)
		sort.Strings(test.y)
		if !reflect.DeepEqual(y, test.y) {
			t.Fatalf("mis-match: %v %v", y, test.y)
		}
	}
}

func BenchmarkTree_Insert_Char(b *testing.B) {
	r := New()
	for i := 0; i < b.N; i++ {
		k := NewCharKey(randomString(8))
		r.Insert(k, i)
	}
}

func BenchmarkTree_Insert_String(b *testing.B) {
	r := New()
	for i := 0; i < b.N; i++ {
		k := NewStringSliceKey(randomStringSlice(8, 8))
		r.Insert(k, i)
	}
}

func BenchmarkTree_Get_Char(b *testing.B) {
	b.StopTimer()
	r := New()
	var keys []Key
	for i := 0; i < b.N; i++ {
		k := NewCharKey(randomString(8))
		r.Insert(k, i)
		keys = append(keys, k)
	}
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		r.Get(keys[i])
	}
}

func BenchmarkTree_Get_String(b *testing.B) {
	b.StopTimer()
	r := New()
	var keys []Key
	for i := 0; i < b.N; i++ {
		k := NewStringSliceKey(randomStringSlice(8, 8))
		r.Insert(k, i)
		keys = append(keys, k)
	}
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		r.Get(keys[i])
	}
}

func BenchmarkTree_Get_Glob(b *testing.B) {
	b.StopTimer()
	r := New()
	var keys []Key
	for i := 0; i < b.N; i++ {
		k := NewGlobSliceKey(randomStringSlice(8, 8))
		r.Insert(k, i)
		keys = append(keys, k)
	}
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		r.Get(keys[i])
	}
}
