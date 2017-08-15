package http

import (
	"context"
	"errors"
	"net/http"
	"sync/atomic"

	"github.com/vegertar/mux/x"
)

// ErrResponseWritten resulted from writting a written response.
var ErrResponseWritten = errors.New("response has been written")

// ResponseWriter implements the `http.ResponseWriter` interface and can check if HTTP response is written.
type ResponseWriter struct {
	http.ResponseWriter
	written int32
}

// Write implements the `http.ResponseWriter` interface.
func (p *ResponseWriter) Write(b []byte) (int, error) {
	defer atomic.CompareAndSwapInt32(&p.written, 0, 1)
	return p.ResponseWriter.Write(b)
}

// WriteHeader implements the `http.ResponseWriter` interface.
func (p *ResponseWriter) WriteHeader(code int) {
	defer atomic.CompareAndSwapInt32(&p.written, 0, 1)
	p.ResponseWriter.WriteHeader(code)
}

// Written returns if the HTTP response is occured.
func (p *ResponseWriter) Written() bool {
	return atomic.LoadInt32(&p.written) == 1
}

// MultiHandler is a wrapper of multiple HTTP handlers.
type MultiHandler []http.Handler

// ServeHTTP implements the `http.Handler` interface.
func (m MultiHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	writer, ok := w.(*ResponseWriter)
	if !ok {
		writer = &ResponseWriter{
			ResponseWriter: w,
		}
	}

	for _, h := range m {
		if writer.Written() {
			break
		}
		h.ServeHTTP(writer, r)
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
)

var (
	notFound = http.NotFoundHandler()
)
