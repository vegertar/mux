package dns

// Middleware is a DNS middleware interface.
// GenerateHandler should generate a new `Handler` from an existed `Handler`.
type Middleware interface {
	GenerateHandler(Handler) Handler
}

// MiddlewareFunc is an adapter to allow the use of ordinary functions as Middleware.
type MiddlewareFunc func(Handler) Handler

// GenerateHandler implements the `Middleware` interface.
func (f MiddlewareFunc) GenerateHandler(h Handler) Handler {
	return f(h)
}
