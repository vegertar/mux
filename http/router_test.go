package http

import (
	"net/http"
	"reflect"
	"strconv"
	"testing"
	"time"

	"github.com/vegertar/mux/x"
)

func TestRouter_HandleFunc(t *testing.T) {
	router := NewRouter()

	for i := 0; i < 2; i++ {
		closer, err := router.HandleFunc(Route{}, func(http.ResponseWriter, *http.Request) {})
		if err != nil {
			t.Fatal(err)
		}
		closer()
	}

	router.DisableDupRoute = true
	closer, err := router.HandleFunc(Route{}, func(http.ResponseWriter, *http.Request) {})
	if err != nil {
		t.Fatal(err)
	}
	defer closer()

	_, err = router.HandleFunc(Route{}, func(http.ResponseWriter, *http.Request) {})
	if err != x.ErrExistedRoute {
		t.Fatal("expected", x.ErrExistedRoute, "got", err)
	}
}

func TestRouter_HandleFuncParallel(t *testing.T) {
	router := NewRouter()
	for i := 0; i < 100; i++ {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			t.Parallel()
			closer, err := router.HandleFunc(Route{}, func(http.ResponseWriter, *http.Request) {})
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
		closer, err := router.UseFunc(Route{}, func(h http.Handler) http.Handler { return h })
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
			closer, err := router.UseFunc(Route{}, func(h http.Handler) http.Handler { return h })
			if err != nil {
				t.Fatal(err)
			}
			time.Sleep(time.Millisecond * 50)
			closer()
		})
	}
}

func TestRouter_ServeHTTP(t *testing.T) {
	routes := []Route{
		{},
		{Path: "/"},
		{Path: "/v1"},
		{Path: "/v1/x"},
		{Path: "/v1/*"},
		{Path: "/v[2-3]"},
		{Path: "/v4/**/x"},
		{Path: "/v4/*/**/x"},
	}

	router := NewRouter()
	for _, route := range routes {
		_, err := router.HandleFunc(route, func(s string) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				w.Header().Add("Y", s)
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
		{Route{Path: "/"}, []string{
			"* *://**/",
			"* *://**/**",
		}},
		{Route{Path: "/v1"}, []string{
			"* *://**/v1",
			"* *://**/**",
		}},
		{Route{Path: "/v1/"}, []string{
			"* *://**/v1/*",
			"* *://**/**",
		}},
		{Route{Path: "/v1/x"}, []string{
			"* *://**/v1/x",
			"* *://**/v1/*",
			"* *://**/**",
		}},
		{Route{Path: "/v1/y"}, []string{
			"* *://**/v1/*",
			"* *://**/**",
		}},
		{Route{Path: "/hello"}, []string{
			"* *://**/**",
		}},
		{Route{Path: "/v2"}, []string{
			"* *://**/v[2-3]",
			"* *://**/**",
		}},
		{Route{Path: "/v3"}, []string{
			"* *://**/v[2-3]",
			"* *://**/**",
		}},
		{Route{Path: "/v4"}, []string{
			"* *://**/**",
		}},
		{Route{Path: "/v4/x"}, []string{
			"* *://**/v4/**/x",
			"* *://**/**",
		}},
		{Route{Path: "/v4/1/x"}, []string{
			"* *://**/v4/*/**/x",
			"* *://**/v4/**/x",
			"* *://**/**",
		}},
	}

	for i, c := range cases {
		c.x.UseLiteral = true
		if c.x.Scheme == "" {
			c.x.Scheme = "http"
		}
		if c.x.Method == "" {
			c.x.Method = "GET"
		}
		if c.x.Host == "" {
			c.x.Host = "localhost"
		}
		if c.x.Path == "" {
			c.x.Path = "/"
		}

		request, err := http.NewRequest(c.x.Method, c.x.Path, nil)
		if err != nil {
			t.Fatal(err)
		}
		request.URL.Scheme = c.x.Scheme
		request.Host = c.x.Host

		w := newHeaderWriter()
		router.ServeHTTP(w, request)
		y := w.Header()["Y"]
		if !reflect.DeepEqual(y, c.y) {
			t.Errorf("bad case %d: expected %v, got %v", i+1, c.y, y)
		}
	}
}

func BenchmarkMatch(b *testing.B) {
	router := NewRouter()
	handler := func(w http.ResponseWriter, r *http.Request) {}
	router.HandleFunc(Route{}, handler)

	c := Route{Path: "/v1/anything", UseLiteral: true}
	for i := 0; i < b.N; i++ {
		router.Match(c)
	}
}

func BenchmarkMux(b *testing.B) {
	router := NewRouter()
	handler := func(w http.ResponseWriter, r *http.Request) {}
	router.HandleFunc(Route{}, handler)

	request, _ := http.NewRequest("GET", "/v1/anything", nil)
	for i := 0; i < b.N; i++ {
		router.ServeHTTP(nil, request)
	}
}

type headerWriter struct {
	h http.Header
}

func (p *headerWriter) Header() http.Header {
	return p.h
}

func (p *headerWriter) Write(b []byte) (int, error) {
	return len(b), nil
}

func (p *headerWriter) WriteHeader(int) {
	return
}

func newHeaderWriter() *headerWriter {
	return &headerWriter{
		h: make(http.Header),
	}
}
