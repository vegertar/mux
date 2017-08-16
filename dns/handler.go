package dns

import (
	"context"
	"errors"

	"github.com/miekg/dns"
	"github.com/vegertar/mux/x"
)

// A ResponseWriter interface is used by a DNS handler to construct an DNS response.
type ResponseWriter interface {
	Header() *dns.MsgHdr
	Answer(...dns.RR)
	Ns(...dns.RR)
	Extra(...dns.RR)
	WriteMsg(*dns.Msg) error
}

type responseWriter struct {
	dns.ResponseWriter
	msg     dns.Msg
	written bool
}

func (p *responseWriter) Header() *dns.MsgHdr {
	return &p.msg.MsgHdr
}

func (p *responseWriter) Answer(r ...dns.RR) {
	p.msg.Answer = append(p.msg.Answer, r...)
}

func (p *responseWriter) Ns(r ...dns.RR) {
	p.msg.Ns = append(p.msg.Ns, r...)
}

func (p *responseWriter) Extra(r ...dns.RR) {
	p.msg.Extra = append(p.msg.Extra, r...)
}

func (p *responseWriter) WriteMsg(msg *dns.Msg) error {
	if msg != nil {
		if msg.Id != 0 {
			p.msg.Id = msg.Id
		}
		if msg.Opcode != 0 {
			p.msg.Opcode = msg.Opcode
		}
		if msg.Authoritative {
			p.msg.Authoritative = msg.Authoritative
		}
		if msg.Truncated {
			p.msg.Truncated = msg.Truncated
		}
		if msg.RecursionDesired {
			p.msg.RecursionDesired = msg.RecursionDesired
		}
		if msg.RecursionAvailable {
			p.msg.RecursionAvailable = msg.RecursionAvailable
		}
		if msg.Zero {
			p.msg.Zero = msg.Zero
		}
		if msg.AuthenticatedData {
			p.msg.AuthenticatedData = msg.AuthenticatedData
		}
		if msg.CheckingDisabled {
			p.msg.CheckingDisabled = msg.CheckingDisabled
		}
		if len(msg.Question) > 0 {
			p.msg.Question = make([]dns.Question, 1)
			p.msg.Question[0] = msg.Question[0]
		}
		if len(msg.Answer) > 0 {
			p.msg.Answer = append(p.msg.Answer, msg.Answer...)
		}
		if len(msg.Ns) > 0 {
			p.msg.Ns = append(p.msg.Ns, msg.Ns...)
		}
		if len(msg.Extra) > 0 {
			p.msg.Extra = append(p.msg.Extra, msg.Extra...)
		}
	}

	if p.ResponseWriter != nil {
		if p.written {
			return ErrResponseWritten
		}

		p.written = true
		p.msg.Response = true
		p.msg.Compress = true

		return p.ResponseWriter.WriteMsg(&p.msg)
	}

	return nil
}

// A Request represents a DNS request received by a server.
type Request struct {
	*dns.Msg

	ctx context.Context
}

// Context returns the request's context. To change the context, use WithContext.
func (r *Request) Context() context.Context {
	if r.ctx != nil {
		return r.ctx
	}
	return context.Background()
}

// WithContext returns a shallow copy of r with its context changed to ctx.
// The provided ctx must be non-nil.
func (r *Request) WithContext(ctx context.Context) *Request {
	if ctx == nil {
		panic("nil context")
	}
	r2 := new(Request)
	*r2 = *r
	r2.ctx = ctx
	return r2
}

// A Handler responds to a DNS request.
type Handler interface {
	ServeDNS(ResponseWriter, *Request)
}

// The HandlerFunc type is an adapter to allow the use of ordinary functions as DNS handlers.
type HandlerFunc func(ResponseWriter, *Request)

// ServeDNS implements `Handler` interface.
func (f HandlerFunc) ServeDNS(w ResponseWriter, r *Request) {
	f(w, r)
}

// MultiHandler is a wrapper of multiple DNS handlers.
type MultiHandler []Handler

// ServeDNS implements `Handler` interface.
func (m MultiHandler) ServeDNS(w ResponseWriter, r *Request) {
	for _, h := range m {
		h.ServeDNS(w, r)
	}
}

func newMultiHandler(handler ...interface{}) MultiHandler {
	m := make([]Handler, 0, len(handler))
	for _, v := range handler {
		if v != nil {
			m = append(m, v.(Handler))
		}
	}
	if len(m) == 0 {
		return nil
	}
	return MultiHandler(m)
}

func newHandlerFromLabels(route x.Route, labels []*x.Label) Handler {
	var (
		h Handler = RefusedErrorHandler

		handlers   []interface{}
		middleware []interface{}
	)

	if len(labels) > 0 {
		// extracts request variables
		middleware = append(middleware, getVars(route, labels[0]))

		for _, label := range labels {
			handlers = append(handlers, label.Handler...)
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
		h = NameErrorHandler
	}
	return h
}

func getVars(route x.Route, label *x.Label) Middleware {
	return MiddlewareFunc(func(h Handler) Handler {
		var varsValue VarsValue

		nameKey := label.Key
		if label.Node != nil {
			nameKey = label.Node.Up().Node.Up().Key
		}
		varsValue.Name = append(varsValue.Name, nameKey.StringWith("."))
		for _, k := range nameKey.Capture(route[0]) {
			varsValue.Name = append(varsValue.Name, k.StringWith("."))
		}

		return HandlerFunc(func(w ResponseWriter, r *Request) {
			h.ServeDNS(w, r.WithContext(context.WithValue(r.Context(), varsKey, varsValue)))
		})
	})
}

// Vars returns the route variables for the current request.
func Vars(r *Request) VarsValue {
	if v := r.Context().Value(varsKey); v != nil {
		return v.(VarsValue)
	}
	return VarsValue{}
}

type (
	contextKey int

	// VarsValue is the value of positional patterns.
	VarsValue struct {
		// Name is the value of host patterns in which [0] is the entire pattern, [1] is the first field, etc.
		Name []string
	}
)

const (
	varsKey contextKey = iota
)

// ErrorHandler responses a given code to client.
type ErrorHandler int

// ServeDNS implements `Handler` interface.
func (e ErrorHandler) ServeDNS(w ResponseWriter, r *Request) {
	w.Header().Rcode = int(e)
	w.WriteMsg(r.Msg)
}

var (
	// ErrResponseWritten resulted from writting a written response.
	ErrResponseWritten = errors.New("response has been written")

	// NoErrorHandler responses `dns.RcodeSuccess`.
	NoErrorHandler = ErrorHandler(dns.RcodeSuccess)
	// NameErrorHandler responses `dns.RcodeNameError`.
	NameErrorHandler = ErrorHandler(dns.RcodeNameError)
	// FormatErrorHandler responses `dns.RcodeFormatError`.
	FormatErrorHandler = ErrorHandler(dns.RcodeFormatError)
	// RefusedErrorHandler responses `dns.RcodeRefused`.
	RefusedErrorHandler = ErrorHandler(dns.RcodeRefused)
	// FailureErrorHandler responses `dns.RcodeServerFailure`.
	FailureErrorHandler = ErrorHandler(dns.RcodeServerFailure)
)
