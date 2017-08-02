package dns

import (
	"context"
	"errors"
	"sync"

	"github.com/miekg/dns"
	"github.com/vegertar/mux/x"
)

type ResponseWriter interface {
	Header() *dns.MsgHdr
	Answer(...dns.RR)
	Ns(...dns.RR)
	Extra(...dns.RR)
	WriteMsg(*dns.Msg) error
	Writer() dns.ResponseWriter
}

func NewResponseWriter(w dns.ResponseWriter) ResponseWriter {
	return &responseWriter{
		ResponseWriter: w,
	}
}

type responseWriter struct {
	dns.ResponseWriter
	msg     dns.Msg
	written bool
	mu      sync.Mutex
}

func (p *responseWriter) Header() *dns.MsgHdr {
	return &p.msg.MsgHdr
}

func (p *responseWriter) Answer(r ...dns.RR) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.msg.Answer = append(p.msg.Answer, r...)
}

func (p *responseWriter) Ns(r ...dns.RR) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.msg.Ns = append(p.msg.Ns, r...)
}

func (p *responseWriter) Extra(r ...dns.RR) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.msg.Extra = append(p.msg.Extra, r...)
}

func (p *responseWriter) WriteMsg(msg *dns.Msg) error {
	p.mu.Lock()
	defer p.mu.Unlock()

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

		for _, r := range msg.Answer {
			p.msg.Answer = append(p.msg.Answer, r)
		}

		for _, r := range msg.Ns {
			p.msg.Ns = append(p.msg.Ns, r)
		}

		for _, r := range msg.Extra {
			p.msg.Extra = append(p.msg.Extra, r)
		}
	}

	if p.ResponseWriter != nil {
		if p.written {
			return ErrMsgWritten
		}

		p.written = true
		p.msg.Response = true
		p.msg.Compress = true

		return p.ResponseWriter.WriteMsg(&p.msg)
	}

	return nil
}

func (p *responseWriter) Writer() dns.ResponseWriter {
	return p.ResponseWriter
}

type Request struct {
	*dns.Msg

	ctx context.Context
}

func (r *Request) Context() context.Context {
	if r.ctx != nil {
		return r.ctx
	}
	return context.Background()
}

func (r *Request) WithContext(ctx context.Context) *Request {
	if ctx == nil {
		panic("nil context")
	}
	r2 := new(Request)
	*r2 = *r
	r2.ctx = ctx
	return r2
}

type Handler interface {
	ServeDNS(ResponseWriter, *Request)
}

type HandlerFunc func(ResponseWriter, *Request)

func (f HandlerFunc) ServeDNS(w ResponseWriter, r *Request) {
	f(w, r)
}

type MultiHandler []Handler

func (m MultiHandler) ServeDNS(w ResponseWriter, r *Request) {
	for _, h := range m {
		h.ServeDNS(w, r)
	}
}

func newMultiHandler(handler ...interface{}) MultiHandler {
	m := make([]Handler, 0, len(handler))
	for _, v := range handler {
		m = append(m, v.(Handler))
	}
	return MultiHandler(m)
}

func newHandlerFromLabels(labels []x.Label) Handler {
	var (
		h Handler = RefusedErrorHandler

		handlers []interface{}
		middleware []interface{}
	)

	for _, label := range labels {
		handlers = append(handlers, label.Handler...)
		middleware = append(middleware, label.Middleware...)
	}
	if len(handlers) > 0 {
		h = newMultiHandler(handlers...)
	}

	for i := range middleware {
		h = middleware[len(middleware)-1-i].(Middleware).GenerateHandler(h)
	}

	if h == nil {
		h = NameErrorHandler
	}
	return h
}

type ErrorHandler int

func (e ErrorHandler) ServeDNS(w ResponseWriter, r *Request) {
	w.Header().Rcode = int(e)
	w.WriteMsg(r.Msg)
}

var (
	ErrMsgWritten = errors.New("message has been written")

	NoErrorHandler      = ErrorHandler(dns.RcodeSuccess)
	NameErrorHandler    = ErrorHandler(dns.RcodeNameError)
	FormatErrorHandler  = ErrorHandler(dns.RcodeFormatError)
	RefusedErrorHandler = ErrorHandler(dns.RcodeRefused)
	FailureErrorHandler = ErrorHandler(dns.RcodeServerFailure)
)
