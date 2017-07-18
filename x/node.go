package x

// Node defines a interface to add, delete, match and iterate a router.
// As the result of `Router` wrapper, a `Node` doesn't need to be implemented concurrent safety.
type Node interface {
	// Make should make all necessary nodes and labels while matching through the given route.
	// If successful, it returns a label matched the last component of route.
	Make(Route) (*Label, error)

	// Delete deletes a label.
	Delete(*Label)

	// Leaves returns all labels has nil `Down` node.
	Leaves() []*Label

	// Get returns a leaf label matched given route exactly.
	Get(Route) *Label

	// Up returns a parent label.
	Up() *Label

	// Empty returns if it is empty.
	Empty() bool

	// Match returns a label matched by given route.
	// The returned label should contain all associated handles and middlewares by order
	// while matching from top to down.
	Match(Route) Label
}
