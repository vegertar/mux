package main

import (
	"context"
	"flag"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
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

	httpRouter.HandleFunc(httpMux.Route{Path:"/pprof/*"}, func(w http.ResponseWriter, r *http.Request) {

	})

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	for sig := range c {
		log.Println(sig.String())
		break
	}
}
