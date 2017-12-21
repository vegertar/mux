# Mux

Mux implements a dynamic DNS/HTTP router which can be safely adding or deleting handlers at any time.

*This package is still under development.*

## Install

With a standard Go toolchain:

```sh
go get -u github.com/vegertar/mux
```

## Examples

```go
package main

import (
	"context"
	"flag"
	"log"
	"net"
	"net/http"
	"net/http/pprof"
	"os"
	"os/signal"
	"strconv"
	"sync"

	"github.com/miekg/dns"
	dnsMux "github.com/vegertar/mux/dns"
	httpMux "github.com/vegertar/mux/http"
)

var (
	dnsRouter  = dnsMux.NewRouter()
	httpRouter = httpMux.NewRouter()
)

func main() {
	flag.Parse()

	var wg sync.WaitGroup
	defer wg.Wait()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	{
		// setup DNS

		dnsAddr := flag.Arg(0)
		if dnsAddr == "" {
			dnsAddr = ":53"
		}
		dnsListener, err := net.ListenPacket("udp", dnsAddr)
		if err != nil {
			log.Fatal(err)
		}
		log.Println("DNS is listening on", dnsListener.LocalAddr())

		dnsServer := &dns.Server{
			PacketConn: dnsListener,
			Handler:    dnsRouter.ServeFunc(ctx),
		}

		wg.Add(1)
		go func() {
			defer wg.Done()
			<-ctx.Done()
			dnsServer.Shutdown()
		}()
		go dnsServer.ActivateAndServe()
	}

	{
		// setup HTTP

		httpListener, err := net.Listen("tcp", ":0")
		if err != nil {
			log.Fatal(err)
		}
		httpAddr := httpListener.Addr()
		log.Println("HTTP is listening on", httpAddr)

		_, port, _ := net.SplitHostPort(httpAddr.String())
		portNo, _ := strconv.Atoi(port)

		srv := new(dns.SRV)
		srv.Hdr.Name = "_http._tcp.example.com."
		srv.Hdr.Rrtype = dns.TypeSRV
		srv.Hdr.Class = dns.ClassINET

		srv.Port = uint16(portNo)
		srv.Target = "localhost."

		a := new(dns.A)
		a.Hdr.Name = srv.Target
		a.Hdr.Rrtype = dns.TypeA
		a.Hdr.Class = dns.ClassINET
		a.A = net.ParseIP("127.0.0.1")

		dnsRouter.HandleFunc(dnsMux.Route{
			Name: "**.example.com.",
			Type: "SRV",
		}, func(w dnsMux.ResponseWriter, r *dnsMux.Request) {
			w.Answer(srv)
		})

		dnsRouter.HandleFunc(dnsMux.Route{
			Name: srv.Target,
			Type: "A",
		}, func(w dnsMux.ResponseWriter, r *dnsMux.Request) {
			w.Answer(a)
		})

		httpServer := &http.Server{
			Handler: httpRouter,
		}

		wg.Add(1)
		go func() {
			defer wg.Done()
			<-ctx.Done()
			httpServer.Shutdown(context.Background())
		}()
		go httpServer.Serve(httpListener)
	}

	httpRouter.HandleFunc(httpMux.Route{Path: "/pprof/*"}, func(w http.ResponseWriter, r *http.Request) {
		switch httpMux.Vars(r).Path[1] {
		case "heap":
			pprof.Handler("heap").ServeHTTP(w, r)
		case "goroutine":
			pprof.Handler("goroutine").ServeHTTP(w, r)
		case "block":
			pprof.Handler("block").ServeHTTP(w, r)
		case "threadcreate":
			pprof.Handler("threadcreate").ServeHTTP(w, r)
		case "cmdline":
			pprof.Cmdline(w, r)
		case "profile":
			pprof.Profile(w, r)
		case "symbol":
			pprof.Symbol(w, r)
		case "trace":
			pprof.Trace(w, r)
		default:
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			pprof.Index(w, r)
		}
	})

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	for sig := range c {
		log.Println(sig.String())
		break
	}
}
```

Building with `go get github.com/vegertar/mux/muxexample` and running with `muxexample :35353`, then use `dig` to lookup `SRV` record, the results shown below.

```sh
$ dig @127.0.0.1 -p 35353 example.com SRV

; <<>> DiG 9.10.3-P4-Ubuntu <<>> @127.0.0.1 -p 35353 example.com SRV
; (1 server found)
;; global options: +cmd
;; Got answer:
;; ->>HEADER<<- opcode: QUERY, status: NOERROR, id: 51354
;; flags: qr rd ad; QUERY: 1, ANSWER: 1, AUTHORITY: 0, ADDITIONAL: 2
;; WARNING: recursion requested but not available

;; OPT PSEUDOSECTION:
; EDNS: version: 0, flags:; udp: 4096
;; QUESTION SECTION:
;example.com.                   IN      SRV

;; ANSWER SECTION:
_http._tcp.example.com. 0       IN      SRV     0 0 43593 localhost.

;; ADDITIONAL SECTION:
localhost.              0       IN      A       127.0.0.1

;; Query time: 1 msec
;; SERVER: 127.0.0.1#35353(127.0.0.1)
;; WHEN: Thu Dec 21 13:48:12 CST 2017
;; MSG SIZE  rcvd: 96
```

## Performance

This mux implementation for DNS and HTTP should consider as a client-side library, e.g. be used for outgoing proxy. Since the underlying data structure isn't designing for huge traffic scenes, the results of benchmark as follows.

For `http`:

```sh
BenchmarkMatch-8                  300000              5321 ns/op
BenchmarkMux-8                    200000              5939 ns/op
BenchmarkParallelMux-8           1000000              2410 ns/op
```

For `dns`:

```sh
BenchmarkMux-8                    300000              4879 ns/op
BenchmarkParallelMux-8           1000000              1836 ns/op
```
