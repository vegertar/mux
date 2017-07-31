package x

// Value is a payload of handles and middlewares.
// Also, an Value carries the current node and a down stream node.
type Value struct {
	Handler    []interface{}
	Middleware []interface{}
	Node       Node
	Down       Node
}

// Root returns up to the toppest root node.
func (p *Value) Root() Node {
	var node Node

	for v := p; v != nil && v.Node != nil; {
		node = v.Node
		v = node.Up()
	}

	return node
}
