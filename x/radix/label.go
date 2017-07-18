package radix

// Label is the minimum comparing unit in the tree.
type Label interface {
	// Compare returns -1, 0, 1 when this label less than, equal to, greater than other bytes respectively.
	Compare(other []byte) int
	// Bytes returns the []byte representation.
	Bytes() []byte
}
