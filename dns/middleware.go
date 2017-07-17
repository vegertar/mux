package dns

type Middleware interface {
	GenerateHandler(Handler) Handler
}

type MiddlewareFunc func(Handler) Handler

func (f MiddlewareFunc) GenerateHandler(h Handler) Handler {
	return f(h)
}
