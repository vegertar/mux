package http

import (
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
	if p.Written() {
		return 0, ErrResponseWritten
	}

	defer atomic.StoreInt32(&p.written, 1)
	return p.ResponseWriter.Write(b)
}

// WriteHeader implements the `http.ResponseWriter` interface.
func (p *ResponseWriter) WriteHeader(code int) {
	if p.Written() {
		return
	}

	defer atomic.StoreInt32(&p.written, 1)
	p.ResponseWriter.WriteHeader(code)
}

// Written returns if the HTTP response is occured.
func (p *ResponseWriter) Written() bool {
	return atomic.LoadInt32(&p.written) == 1
}

// MultiHandler is a wrapper of multiple http handlers.
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
		m = append(m, v.(http.Handler))
	}
	return MultiHandler(m)
}

func newHandlerFromLabel(label x.Label) http.Handler {
	h := notFound
	if len(label.Handler) != 0 {
		h = newMultiHandler(label.Handler...)
	}

	middleware := label.Middleware
	for i := range middleware {
		h = middleware[len(middleware)-1-i].(Middleware).GenerateHandler(h)
	}

	if h == nil {
		h = notFound
	}
	return h
}

var (
	notFound = http.NotFoundHandler()
)
