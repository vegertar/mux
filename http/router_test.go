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

func TestRouter_Match(t *testing.T) {
	routes := []Route{
		{},
		{Path: "/"},
		{Path: "/v1"},
		{Path: "/v1/x"},
		{Path: "/v1/*"},
	}

	router := NewRouter()
	for _, route := range routes {
		router.HandleFunc(route, func(s string) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				w.Header().Add("Y", s)
			}
		}(route.String()))
	}

	cases := []struct {
		x Route
		y []string
	}{
		{Route{Path: "/"}, []string{
			"* *://*/",
			"* *://**",
		}},
		{Route{Path: "/v1"}, []string{
			"* *://*/v1",
			"* *://**",
		}},
		{Route{Path: "/v1/"}, []string{
			"* *://*/v1/*",
			"* *://**",
		}},
		{Route{Path: "/v1/x"}, []string{
			"* *://*/v1/x",
			"* *://*/v1/*",
			"* *://**",
		}},
		{Route{Path: "/v1/y"}, []string{
			"* *://*/v1/*",
			"* *://**",
		}},
		{Route{Path: "/hello"}, []string{
			"* *://**",
		}},
	}

	for i, c := range cases {
		c.x.UseLiteral = true
		if c.x.Scheme == "" {
			c.x.Scheme = "http"
		}
		if c.x.Host == "" {
			c.x.Host = "localhost"
		}
		if c.x.Path == "" {
			c.x.Path = "/"
		}

		w := newHeaderWriter()
		router.Match(c.x).ServeHTTP(w, nil)
		y := w.Header()["Y"]
		if !reflect.DeepEqual(y, c.y) {
			t.Errorf("bad case %d: expected %v, got %v", i + 1, c.y, y)
		}
	}
}

func BenchmarkMux(b *testing.B) {
	router := NewRouter()
	handler := func(w http.ResponseWriter, r *http.Request) {}
	router.HandleFunc(Route{Path: "/v1/*"}, handler)

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
