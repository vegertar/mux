package http

import (
	"net/http"
)

// Middleware is an HTTP middleware interface.
// GenerateHandler should generate a new `http.Handler` from an existed `http.Handler`.
type Middleware interface {
	GenerateHandler(http.Handler) http.Handler
}

// MiddlewareFunc is an adapter to allow the use of ordinary functions as Middleware.
type MiddlewareFunc func(http.Handler) http.Handler

// GenerateHandler implements the `Middleware` interface.
func (f MiddlewareFunc) GenerateHandler(h http.Handler) http.Handler {
	return f(h)
}
