package radix

import (
	"math/rand"
//	"reflect"
//	"sort"
//	"testing"
)

func randomByteLabel() Label {
	return StringLabel(randomString(1))
}

func randomString(n int) string {
	b := make([]byte, 0, n)
	for i := 0; i < n; i++ {
		b = append(b, byte(rand.Intn(26)+65))
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

//
//func TestTree(t *testing.T) {
//	var min, max string
//	input := randomStringSlice(1000, 20)
//	sort.Strings(input)
//
//	r := New()
//	for i := 0; i < len(input); i++ {
//		s := input[i]
//		if s < min || i == 0 {
//			min = s
//		}
//		if s > max || i == 0 {
//			max = s
//		}
//		r.Insert(NewCharKey(s), i)
//	}
//
//	if r.Len() != len(input) {
//		t.Fatalf("bad tree length: expected %d, got %d", len(input), r.Len())
//	}
//
//	// Check walking in order
//	r.Walk(func(k Key, v interface{}) bool {
//		i := v.(int)
//		s := input[i]
//
//		if s != k.String() {
//			t.Fatalf("bad key: expected %v, got %v", s, k)
//		}
//		return false
//	})
//
//	for i, k := range input {
//		out, ok := r.Get(NewCharKey(k))
//		if !ok {
//			t.Fatalf("missing key: %v", k)
//		}
//		if out != i {
//			t.Fatalf("value mis-match: %v %v", out, i)
//		}
//	}
//
//	// Check min and max
//	outMin, _, _ := r.Minimum()
//	if min != outMin.String() {
//		t.Fatalf("bad minimum: %v %v", outMin, min)
//	}
//	outMax, _, _ := r.Maximum()
//	if max != outMax.String() {
//		t.Fatalf("bad maximum: %v %v", outMax, max)
//	}
//
//	for i, k := range input {
//		out, ok := r.Delete(NewCharKey(k))
//		if !ok {
//			t.Fatalf("missing key: %v", k)
//		}
//		if out != i {
//			t.Fatalf("value mis-match: %v %v", out, i)
//		}
//	}
//	if r.Len() != 0 {
//		t.Fatalf("bad length: %v", r.Len())
//	}
//}
//
//func TestRoot(t *testing.T) {
//	r := New()
//	k := NewCharKey("")
//	_, ok := r.Delete(k)
//	if ok {
//		t.Fatalf("bad")
//	}
//	_, ok = r.Insert(k, true)
//	if ok {
//		t.Fatalf("bad")
//	}
//	val, ok := r.Get(k)
//	if !ok || val != true {
//		t.Fatalf("bad: %v", val)
//	}
//	val, ok = r.Delete(k)
//	if !ok || val != true {
//		t.Fatalf("bad: %v", val)
//	}
//}
//
//func TestDelete(t *testing.T) {
//	r := New()
//	keys := []string{"", "A", "AB"}
//
//	for _, key := range keys {
//		r.Insert(NewCharKey(key), true)
//	}
//
//	for _, key := range keys {
//		_, ok := r.Delete(NewCharKey(key))
//		if !ok {
//			t.Fatalf("bad %v", key)
//		}
//	}
//}
//
//func TestLongestPrefix_Char(t *testing.T) {
//	r := New()
//
//	keys := []string{
//		"",
//		"foo",
//		"foobar",
//		"foobarbaz",
//		"foobarbazzip",
//		"foozip",
//	}
//	for _, k := range keys {
//		r.Insert(NewCharKey(k), nil)
//	}
//	if r.Len() != len(keys) {
//		t.Fatalf("bad len: %v %v", r.Len(), len(keys))
//	}
//
//	type exp struct {
//		x string
//		y string
//	}
//	cases := []exp{
//		{"a", ""},
//		{"abc", ""},
//		{"fo", ""},
//		{"foo", "foo"},
//		{"foob", "foo"},
//		{"foobar", "foobar"},
//		{"foobarba", "foobar"},
//		{"foobarbaz", "foobarbaz"},
//		{"foobarbazzi", "foobarbaz"},
//		{"foobarbazzip", "foobarbazzip"},
//		{"foozi", "foo"},
//		{"foozip", "foozip"},
//		{"foozipzap", "foozip"},
//	}
//	for _, test := range cases {
//		x := NewCharKey(test.x)
//		m, _, ok := r.LongestPrefix(x)
//		if !ok {
//			t.Fatalf("no match: %v", test)
//		}
//		if m.String() != test.y {
//			t.Fatalf("mis-match: %v %v", m, test)
//		}
//	}
//}
//
//func TestLongestPrefix_Glob(t *testing.T) {
//	r := New()
//
//	keys := []string{
//		"",
//		"foo",
//		"foo/bar",
//		"foo/*/baz",
//		"foo/bar/baz/zip",
//		"foo/bar/baz/*",
//		"foo/zip*",
//	}
//	for _, k := range keys {
//		x := NewGlobKey(k, "/")
//		old, updated := r.Insert(x, nil)
//		if updated {
//			t.Fatalf("bad updating: %v, %v", old, x)
//		}
//	}
//	if r.Len() != len(keys) {
//		t.Fatalf("bad len: %v %v", r.Len(), len(keys))
//	}
//
//	type exp struct {
//		x string
//		y string
//	}
//	cases := []exp{
//		{"a", ""},
//		{"abc", ""},
//		{"fo", ""},
//		{"foo", "foo"},
//		{"foo/b", "foo"},
//		{"foo/bar", "foo/bar"},
//		{"foo/bar/ba", "foo/bar"},
//		{"foo/bar/baz", "foo/*/baz"},
//		{"foo/bar/baz/zi", "foo/bar/baz/*"},
//		{"foo/bar/baz/zip", "foo/bar/baz/zip"},
//		{"foo/zi", "foo"},
//		{"foo/zip", "foo/zip*"},
//		{"foo/zip1", "foo/zip*"},
//	}
//	for _, test := range cases {
//		x := NewGlobKey(test.x, "/")
//		m, _, ok := r.LongestPrefix(x)
//		if !ok {
//			t.Fatalf("no match: %v", test)
//		}
//		if m.String() != test.y {
//			t.Fatalf("mis-match: %v %v", m, test)
//		}
//	}
//}
//
//func TestWalkPrefix(t *testing.T) {
//	r := New()
//
//	keys := []string{
//		"foobar",
//		"foo/bar/baz",
//		"foo/baz/bar",
//		"foo/zip/zap",
//		"zipzap",
//	}
//	for _, k := range keys {
//		r.Insert(NewCharKey(k), nil)
//	}
//	if r.Len() != len(keys) {
//		t.Fatalf("bad len: %v %v", r.Len(), len(keys))
//	}
//
//	type exp struct {
//		x string
//		y []string
//	}
//	cases := []exp{
//		{
//			"f",
//			[]string{"foobar", "foo/bar/baz", "foo/baz/bar", "foo/zip/zap"},
//		},
//		{
//			"foo",
//			[]string{"foobar", "foo/bar/baz", "foo/baz/bar", "foo/zip/zap"},
//		},
//		{
//			"foob",
//			[]string{"foobar"},
//		},
//		{
//			"foo/",
//			[]string{"foo/bar/baz", "foo/baz/bar", "foo/zip/zap"},
//		},
//		{
//			"foo/b",
//			[]string{"foo/bar/baz", "foo/baz/bar"},
//		},
//		{
//			"foo/ba",
//			[]string{"foo/bar/baz", "foo/baz/bar"},
//		},
//		{
//			"foo/bar",
//			[]string{"foo/bar/baz"},
//		},
//		{
//			"foo/bar/baz",
//			[]string{"foo/bar/baz"},
//		},
//		{
//			"foo/bar/bazoo",
//			[]string{},
//		},
//		{
//			"z",
//			[]string{"zipzap"},
//		},
//	}
//
//	for _, test := range cases {
//		y := []string{}
//		fn := func(k Key, v interface{}) bool {
//			y = append(y, k.String())
//			return false
//		}
//		r.WalkPrefix(NewCharKey(test.x), fn)
//		sort.Strings(y)
//		sort.Strings(test.y)
//		if !reflect.DeepEqual(y, test.y) {
//			t.Fatalf("mis-match: %v %v", y, test.y)
//		}
//	}
//}
//
//func TestWalkPath(t *testing.T) {
//	r := New()
//
//	keys := []string{
//		"foo",
//		"foo/bar",
//		"foo/bar/baz",
//		"foo/baz/bar",
//		"foo/zip/zap",
//		"zipzap",
//	}
//	for _, k := range keys {
//		r.Insert(NewCharKey(k), nil)
//	}
//	if r.Len() != len(keys) {
//		t.Fatalf("bad len: %v %v", r.Len(), len(keys))
//	}
//
//	type exp struct {
//		x string
//		y []string
//	}
//	cases := []exp{
//		{
//			"f",
//			[]string{},
//		},
//		{
//			"foo",
//			[]string{"foo"},
//		},
//		{
//			"foo/",
//			[]string{"foo"},
//		},
//		{
//			"foo/ba",
//			[]string{"foo"},
//		},
//		{
//			"foo/bar",
//			[]string{"foo", "foo/bar"},
//		},
//		{
//			"foo/bar/baz",
//			[]string{"foo", "foo/bar", "foo/bar/baz"},
//		},
//		{
//			"foo/bar/bazoo",
//			[]string{"foo", "foo/bar", "foo/bar/baz"},
//		},
//		{
//			"z",
//			[]string{},
//		},
//	}
//
//	for _, test := range cases {
//		y := []string{}
//		fn := func(k Key, v interface{}) bool {
//			y = append(y, k.String())
//			return false
//		}
//		r.WalkPath(NewCharKey(test.x), fn)
//		sort.Strings(y)
//		sort.Strings(test.y)
//		if !reflect.DeepEqual(y, test.y) {
//			t.Fatalf("mis-match: %v %v", y, test.y)
//		}
//	}
//}
//
//func BenchmarkLongestPrefix(b *testing.B) {
//	b.StopTimer()
//	var keys []Key
//	for i := 0; i < b.N; i++ {
//		k := NewCharKey(randomString(8))
//		keys = append(keys, k)
//	}
//	b.StartTimer()
//	for i := 0; i < b.N; i++ {
//		longestPrefix(keys[i], keys[i])
//	}
//}
//
//func BenchmarkEdges_Search(b *testing.B) {
//	b.StopTimer()
//	var e edges
//	for i := 0; i < b.N; i++ {
//		e = append(e, edge{
//			label: randomByteLabel(),
//		})
//	}
//
//	e.Sort()
//	b.StartTimer()
//	for i := 0; i < b.N; i++ {
//		e.Search(randomByteLabel())
//	}
//}
//
//func BenchmarkTree_LongestPrefix(b *testing.B) {
//	b.StopTimer()
//	r := New()
//	var keys []Key
//	for i := 0; i < b.N; i++ {
//		k := NewCharKey(randomString(8))
//		r.Insert(k, i)
//		keys = append(keys, k)
//	}
//	b.StartTimer()
//	for i := 0; i < b.N; i++ {
//		r.LongestPrefix(keys[i])
//	}
//}
//
//func BenchmarkTree_Insert(b *testing.B) {
//	r := New()
//	for i := 0; i < b.N; i++ {
//		k := NewCharKey(randomString(8))
//		r.Insert(k, i)
//	}
//}
//
//func BenchmarkMap_Insert(b *testing.B) {
//	m := make(map[string]interface{})
//	for i := 0; i < b.N; i++ {
//		m[randomString(8)] = i
//	}
//}
//
//func BenchmarkTree_Get(b *testing.B) {
//	b.StopTimer()
//	r := New()
//	var keys []Key
//	for i := 0; i < b.N; i++ {
//		k := NewCharKey(randomString(8))
//		r.Insert(k, i)
//		keys = append(keys, k)
//	}
//	b.StartTimer()
//	for i := 0; i < b.N; i++ {
//		r.Get(keys[i])
//	}
//}
//
//func BenchmarkMap_Get(b *testing.B) {
//	b.StopTimer()
//	m := make(map[string]interface{})
//	var keys []string
//	for i := 0; i < b.N; i++ {
//		k := randomString(8)
//		m[k] = i
//		keys = append(keys, k)
//	}
//	b.StartTimer()
//	var hitCount int64
//	for i := 0; i < b.N; i++ {
//		if m[keys[i]] != nil {
//			hitCount++
//		}
//	}
//}
