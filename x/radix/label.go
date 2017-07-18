package radix

import "bytes"


// Label is the minimum comparing unit in the tree.
type Label interface {
	// Compare returns -1, 0, 1 when this label less than, equal to, greater than other label respectively.
	Compare(other []byte) int
	// Bytes returns the []byte representation.
	Bytes() []byte
}

type (
	StringKey string
	ByteLabel  byte
)

func (s StringKey) Labels() []Label {
	b := make([]Label, 0, len(s))
	for _, c := range s {
		b = append(b, ByteLabel(c))
	}
	return b
}

func (b ByteLabel) Compare(other []byte) int {
	return bytes.Compare(b.Bytes(), other)
}

func (b ByteLabel) Bytes() []byte {
	return []byte{b}
}
