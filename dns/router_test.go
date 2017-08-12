package dns

import (
	"reflect"
	"strconv"
	"testing"
	"time"

	"github.com/miekg/dns"
	"github.com/vegertar/mux/x"
)

func TestRouter_HandleFunc(t *testing.T) {
	router := NewRouter()

	for i := 0; i < 2; i++ {
		closer, err := router.HandleFunc(Route{}, func(ResponseWriter, *Request) {})
		if err != nil {
			t.Fatal(err)
		}
		closer()
	}

	router.DisableDupRoute = true
	closer, err := router.HandleFunc(Route{}, func(ResponseWriter, *Request) {})
	if err != nil {
		t.Fatal(err)
	}
	defer closer()

	_, err = router.HandleFunc(Route{}, func(ResponseWriter, *Request) {})
	if err != x.ErrExistedRoute {
		t.Fatal("expected", x.ErrExistedRoute, "got", err)
	}
}

func TestRouter_HandleFuncParallel(t *testing.T) {
	router := NewRouter()
	for i := 0; i < 100; i++ {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			t.Parallel()
			closer, err := router.HandleFunc(Route{}, func(ResponseWriter, *Request) {})
			if err != nil {
				t.Fatal(err)
			}
			time.Sleep(time.Millisecond * 50)
			closer()
		})
	}
}

func TestRouter_UseFunc(t *testing.T) {
	router := NewRouter()
	for i := 0; i < 2; i++ {
		closer, err := router.UseFunc(Route{}, func(h Handler) Handler { return h })
		if err != nil {
			t.Fatal(err)
		}
		closer()
	}
}

func TestRouter_UseFuncParallel(t *testing.T) {
	router := NewRouter()
	for i := 0; i < 100; i++ {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			t.Parallel()
			closer, err := router.UseFunc(Route{}, func(h Handler) Handler { return h })
			if err != nil {
				t.Fatal(err)
			}
			time.Sleep(time.Millisecond * 50)
			closer()
		})
	}
}

func TestRouter_ServeDNS(t *testing.T) {
	routes := []Route{
		{},
		{Name: "v1"},
		{Name: "v1.x"},
		{Name: "v1.*"},
		{Name: "v[2-3]"},
		{Name: "v4.**.x"},
		{Name: "v4.*.**.x"},
	}

	router := NewRouter()
	for _, route := range routes {
		_, err := router.HandleFunc(route, func(s string) HandlerFunc {
			return func(w ResponseWriter, r *Request) {
				a := new(dns.A)
				a.Hdr.Name = s
				w.Answer(a)
			}
		}(route.String()))
		if err != nil {
			t.Fatal(err)
		}
	}
	if n := len(router.Routes()); n != len(routes) {
		t.Fatalf("expected %d routes, got %d", len(routes), n)
	}

	cases := []struct {
		x Route
		y []string
	}{
		{Route{Name: "v1"}, []string{
			"v1 A IN",
		}},
		{Route{Name: "v1."}, []string{
			"v1 A IN",
		}},
		{Route{Name: "v1.x"}, []string{
			"v1.x A IN",
		}},
		{Route{Name: "v1.y"}, []string{
			"v1.* A IN",
		}},
	}

	for i, c := range cases {
		c.x.UseLiteral = true

		request := &Request{
			Msg: new(dns.Msg),
		}
		request.SetQuestion(c.x.Name, dns.TypeA)
		w := new(responseWriter)

		router.ServeDNS(w, request)
		var y []string
		for _, rr := range w.msg.Answer {
			y = append(y, rr.Header().Name)
		}
		if !reflect.DeepEqual(y, c.y) {
			t.Errorf("bad case %d: expected %v, got %v", i+1, c.y, y)
		}
	}
}
