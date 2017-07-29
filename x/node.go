package x

import "github.com/vegertar/mux/x/radix"

// Leaf is used to represent search results.
type Leaf struct {
	Key   radix.Key
	Value *Value
}

// Node defines a interface to add, delete, match and iterate a router.
// As the result of `Router` wrapper, a `Node` doesn't need to be implemented concurrent safety.
type Node interface {
	// Make should make all necessary nodes while matching through the given route.
	// If successful, it returns a leaf.
	Make(Route) (*Leaf, error)

	// Delete deletes a leaf.
	Delete(*Leaf)

	// Leaves returns all leaves has nil `Down` node.
	Leaves() []*Leaf

	// Get returns a leaf matched given route exactly.
	Get(Route) *Leaf

	// Up returns a parent label.
	Up() *Label

	// Empty returns if it is empty.
	Empty() bool

	// Match returns a label matched by given route.
	// The returned label should contain all associated handles and middlewares by order
	// while matching from top to down.
	Match(Route) Label
}
