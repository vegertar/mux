package x

// Value is a payload of handlers and middleware.
// Also, an Value carries the current node and a down stream node.
type Value struct {
	Handler    []interface{}
	Middleware []interface{}
	Node       Node
	Down       Node
}
