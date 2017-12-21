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
