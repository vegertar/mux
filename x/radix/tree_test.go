package radix

import (
	"bytes"
	"math/rand"
	"reflect"
	"sort"
	"testing"
)

type bytesLabel []byte

func (self bytesLabel) Compare(other []byte) int {
	return bytes.Compare(self.Bytes(), other)
}

func (self bytesLabel) Bytes() []byte {
	return []byte(self)
}

type bytesArray [][]byte

func (self bytesArray) Len() int { return len(self) }

func (self bytesArray) Less(i, j int) bool { return bytes.Compare(self[i], self[j]) < 0 }

func (self bytesArray) Swap(i, j int) { self[i], self[j] = self[j], self[i] }

func bytesArrayToLabels(values [][]byte) []Label {
	v := make([]Label, 0, len(values))
	for _, s := range values {
		v = append(v, bytesLabel(s))
	}
	return v
}

func labelsToBytesArray(labels []Label) [][]byte {
	v := make([][]byte, 0, len(labels))
	for _, l := range labels {
		v = append(v, l.Bytes())
	}
	return v
}

func bytesToLabels(s []byte) []Label {
	return bytesArrayToLabels(bytes.Split(s, []byte("")))
}

func labelsToBytes(labels []Label) []byte {
	return bytes.Join(labelsToBytesArray(labels), []byte(""))
}

func compareBytesAndLabels(s []byte, labels []Label) int {
	for i := 0; i < len(s) && i < len(labels); i++ {
		a := s[i]
		b := labels[i].Bytes()
		if d := bytes.Compare([]byte{a}, b); d != 0 {
			return d
		}
	}
	d := len(s) - len(labels)
	if d < 0 {
		d = -1
	} else if d > 0 {
		d = 1
	}
	return d
}

func randomByteLabel() Label {
	return bytesLabel([]byte{byte(rand.Intn(26) + 65)})
}

func randomBytes(n int) []byte {
	b := make([]byte, 0, n)
	for i := 0; i < n; i++ {
		b = append(b, byte(rand.Intn(26)+65))
	}
	return b
}

func randomBytesArray(size, unit int) [][]byte {
	values := make([][]byte, 0, size)
	for i := 0; i < size; i++ {
		values = append(values, randomBytes(unit))
	}
	return values
}

func TestEdges_Search(t *testing.T) {
	const n = 100
	values := randomBytesArray(n, 10)
	if len(values) != n {
		t.Fatalf("expected %d random strings, got %d", n, len(values))
	}

	var e edges
	for _, s := range values {
		e = append(e, edge{
			label: bytesLabel(s),
		})
	}

	e.Sort()
	sort.Sort(bytesArray(values))

	for i := range rand.Perm(len(values)) {
		s := values[i]
		a := sort.Search(len(values), func(j int) bool { return bytes.Compare(values[j], s) >= 0 })
		b := e.Search(bytesLabel(s))

		if a != b {
			t.Fatalf("search %s expected %d, got %d", s, a, b)
		}
	}
}

func TestTree(t *testing.T) {
	var min, max []byte
	input := randomBytesArray(1000, 20)
	sort.Sort(bytesArray(input))

	r := New()
	for i := 0; i < len(input); i++ {
		s := input[i]
		if bytes.Compare(s, min) < 0 || i == 0 {
			min = s
		}
		if bytes.Compare(s, max) > 0 || i == 0 {
			max = s
		}
		r.Insert(bytesToLabels(s), i)
	}

	if r.Len() != len(input) {
		t.Fatalf("bad tree length: expected %d, got %d", len(input), r.Len())
	}

	// Check walking in order
	r.Walk(func(k []Label, v interface{}) bool {
		i := v.(int)
		s := input[i]

		if compareBytesAndLabels(s, k) != 0 {
			t.Fatalf("bad key: expected %v, got %v", s, k)
		}
		return false
	})

	for i, k := range input {
		out, ok := r.Get(bytesToLabels(k))
		if !ok {
			t.Fatalf("missing key: %v", k)
		}
		if out != i {
			t.Fatalf("value mis-match: %v %v", out, i)
		}
	}

	// Check min and max
	outMin, _, _ := r.Minimum()
	if compareBytesAndLabels(min, outMin) != 0 {
		t.Fatalf("bad minimum: %v %v", outMin, min)
	}
	outMax, _, _ := r.Maximum()
	if compareBytesAndLabels(max, outMax) != 0 {
		t.Fatalf("bad maximum: %v %v", outMax, max)
	}

	for i, k := range input {
		out, ok := r.Delete(bytesToLabels(k))
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
	k := bytesToLabels([]byte(""))
	_, ok := r.Delete(k)
	if ok {
		t.Fatalf("bad")
	}
	_, ok = r.Insert(k, true)
	if ok {
		t.Fatalf("bad")
	}
	val, ok := r.Get(k)
	if !ok || val != true {
		t.Fatalf("bad: %v", val)
	}
	val, ok = r.Delete(k)
	if !ok || val != true {
		t.Fatalf("bad: %v", val)
	}
}

func TestDelete(t *testing.T) {
	r := New()
	keys := [][]Label{bytesToLabels([]byte("")), bytesToLabels([]byte("A")), bytesToLabels([]byte("AB"))}

	for _, key := range keys {
		r.Insert(key, true)
	}

	for _, key := range keys {
		_, ok := r.Delete(key)
		if !ok {
			t.Fatalf("bad %v", key)
		}
	}
}

func TestLongestPrefix(t *testing.T) {
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
		r.Insert(bytesToLabels([]byte(k)), nil)
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
		x, y := bytesToLabels([]byte(test.x)), []byte(test.y)
		m, _, ok := r.LongestPrefix(x)
		if !ok {
			t.Fatalf("no match: %v", test)
		}
		if compareBytesAndLabels(y, m) != 0 {
			t.Fatalf("mis-match: %v %v", m, test)
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
		r.Insert(bytesToLabels([]byte(k)), nil)
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
		fn := func(k []Label, v interface{}) bool {
			y = append(y, string(labelsToBytes(k)))
			return false
		}
		r.WalkPrefix(bytesToLabels([]byte(test.x)), fn)
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
		r.Insert(bytesToLabels([]byte(k)), nil)
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
		fn := func(k []Label, v interface{}) bool {
			y = append(y, string(labelsToBytes(k)))
			return false
		}
		r.WalkPath(bytesToLabels([]byte(test.x)), fn)
		sort.Strings(y)
		sort.Strings(test.y)
		if !reflect.DeepEqual(y, test.y) {
			t.Fatalf("mis-match: %v %v", y, test.y)
		}
	}
}

func BenchmarkLongestPrefix(b *testing.B) {
	b.StopTimer()
	var keys [][]Label
	for i := 0; i < b.N; i++ {
		k := bytesToLabels(randomBytes(8))
		keys = append(keys, k)
	}
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		longestPrefix(keys[i], keys[i])
	}
}

func BenchmarkEdges_Search(b *testing.B) {
	b.StopTimer()
	var e edges
	for i := 0; i < b.N; i++ {
		e = append(e, edge{
			label: randomByteLabel(),
		})
	}

	e.Sort()
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		e.Search(randomByteLabel())
	}
}

func BenchmarkTree_LongestPrefix(b *testing.B) {
	b.StopTimer()
	r := New()
	var keys [][]Label
	for i := 0; i < b.N; i++ {
		k := bytesToLabels(randomBytes(8))
		r.Insert(k, i)
		keys = append(keys, k)
	}
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		r.LongestPrefix(keys[i])
	}
}

func BenchmarkTree_Insert(b *testing.B) {
	r := New()
	for i := 0; i < b.N; i++ {
		k := bytesToLabels(randomBytes(8))
		r.Insert(k, i)
	}
}

func BenchmarkMap_Insert(b *testing.B) {
	m := make(map[string]interface{})
	for i := 0; i < b.N; i++ {
		b := randomBytes(8)
		m[string(b)] = i
	}
}

func BenchmarkTree_Get(b *testing.B) {
	b.StopTimer()
	r := New()
	var keys [][]Label
	for i := 0; i < b.N; i++ {
		k := bytesToLabels(randomBytes(8))
		r.Insert(k, i)
		keys = append(keys, k)
	}
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		r.Get(keys[i])
	}
}

func BenchmarkMap_Get(b *testing.B) {
	b.StopTimer()
	m := make(map[string]interface{})
	var keys []string
	for i := 0; i < b.N; i++ {
		k := string(randomBytes(8))
		m[k] = i
		keys = append(keys, k)
	}
	b.StartTimer()
	var hitCount int64
	for i := 0; i < b.N; i++ {
		if m[keys[i]] != nil {
			hitCount++
		}
	}
}
