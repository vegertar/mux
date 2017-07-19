package radix

import (
	"testing"
	"bytes"
	"k8s.io/kubernetes/pkg/util/rand"
	"sort"
)

type stringLabel string

func (self stringLabel) Compare(other []byte) int {
	return bytes.Compare(self.Bytes(), other)
}

func (self stringLabel) Bytes() []byte {
	return []byte(self)
}

func randomString(n int) string {
	b := make([]byte, 0, n)
	for i := 0; i < n; i++ {
		b = append(b, byte(rand.Intn(256)))
	}
	return string(b)
}

func randomStrings(size, unit int) []string {
	values := make([]string, 0, size)
	for i := 0; i < size; i++ {
		values = append(values, randomString(unit))
	}
	return values
}

func TestEdges_Search(t *testing.T) {
	const n = 100
	values := randomStrings(n, 10)
	if len(values) != n {
		t.Fatalf("expected %d random strings, got %d", n, len(values))
	}

	var e edges
	for _, s := range values {
		e = append(e, edge{
			label: stringLabel(s),
		})
	}

	e.Sort()
	sort.Strings(values)

	for i := range rand.Perm(len(values)) {
		s := values[i]
		a := sort.SearchStrings(values, s)
		b := e.Search(stringLabel(s))

		if a != b {
			t.Fatalf("search %s expected %d, got %d", s, a, b)
		}
	}
}
