package http

import (
	"context"
	"net/http"

	"github.com/vegertar/mux/x"
)

// MultiHandler is a wrapper of multiple HTTP handlers.
type MultiHandler []http.Handler

// ServeHTTP implements the `http.Handler` interface.
func (m MultiHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	for _, h := range m {
		h.ServeHTTP(w, r)
	}
}

func newMultiHandler(handler ...interface{}) MultiHandler {
	m := make([]http.Handler, 0, len(handler))
	for _, v := range handler {
		if v != nil {
			m = append(m, v.(http.Handler))
		}
	}
	if len(m) == 0 {
		return nil
	}
	return MultiHandler(m)
}

func newHandlerFromLabels(route x.Route, labels []*x.Label) http.Handler {
	var (
		h = notFound

		handlers   []interface{}
		middleware []interface{}
	)

	if len(labels) > 0 {
		// remains the first matched handlers only
		handlers = append(handlers, labels[0].Handler...)
		// extracts request variables
		middleware = append(middleware, getVars(route, labels[0]))
		// adds ordinary middleware
		for _, label := range labels {
			middleware = append(middleware, label.Middleware...)
		}
	}

	if len(handlers) > 0 {
		h = newMultiHandler(handlers...)
	}

	for i := range middleware {
		if m := middleware[len(middleware)-1-i]; m != nil {
			h = m.(Middleware).GenerateHandler(h)
		}
	}

	if h == nil {
		h = notFound
	}
	return h
}

func getVars(route x.Route, label *x.Label) Middleware {
	return MiddlewareFunc(func(h http.Handler) http.Handler {
		var varsValue VarsValue

		pathKey := label.Key
		varsValue.Path = append(varsValue.Path, pathKey.StringWith("/"))
		for _, k := range pathKey.Capture(route[len(route)-1]) {
			varsValue.Path = append(varsValue.Path, k.StringWith("/"))
		}

		hostKey := label.Node.Up().Key
		varsValue.Host = append(varsValue.Host, hostKey.StringWith("."))
		for _, k := range hostKey.Capture(route[len(route)-2]) {
			varsValue.Host = append(varsValue.Host, k.StringWith("."))
		}

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), varsKey, varsValue)))
		})
	})
}

// Vars returns the route variables for the current request.
func Vars(r *http.Request) VarsValue {
	if v := r.Context().Value(varsKey); v != nil {
		return v.(VarsValue)
	}
	return VarsValue{}
}

type (
	contextKey int

	// VarsValue is the value of positional patterns.
	VarsValue struct {
		// Host is the value of host patterns in which [0] is the entire pattern, [1] is the first field, etc.
		Host []string
		// Path is the value of path patterns in which [0] is the entire pattern, [1] is the first field, etc.
		Path []string
	}
)

const (
	varsKey contextKey = iota

	// RouterContextKey is a context key. It can be used in HTTP
	// handlers with context.WithValue to access the router that
	// routine the handler. The associated value will be of
	// type *Router.
	RouterContextKey
)

var (
	notFound = http.NotFoundHandler()
)
