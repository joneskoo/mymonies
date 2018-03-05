package main

import (
	"flag"
	"log"
	"net"
	"net/http"
	"os"

	"github.com/joneskoo/mymonies/pkg/middleware"
	"github.com/joneskoo/mymonies/pkg/mymoniesserver"
	"github.com/joneskoo/mymonies/pkg/rpc/mymonies"
)

func main() {
	conn := flag.String("postgres", "database=mymonies", "PostgreSQL connection string")
	listen := flag.String("listen", defaultListen(), "HTTP server listen address")
	flag.Parse()

	logger := log.New(os.Stdout, "[mymonies] ", log.Lshortfile)
	logger.Println("Listening on http://" + *listen)

	server, err := mymoniesserver.New(*conn, logger)
	if err != nil {
		logger.Fatal(err)
	}
	h := handler(server)
	logger.Fatal(http.ListenAndServe(*listen, h))
}

func defaultListen() string {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8000"
	}
	return net.JoinHostPort("127.0.0.1", port)

}

func handler(server mymonies.Mymonies) http.Handler {
	mux := http.NewServeMux()

	// Twirp RPC handler with prometheus metrics
	// hooks := prometheus.NewServerHooks(nil)
	twirpHandler := mymonies.NewMymoniesServer(server, nil)
	mux.Handle(mymonies.MymoniesPathPrefix, twirpHandler)

	// Prometheus metrics endpoint
	// mux.Handle("/metrics", promhttp.Handler())

	// Static file server
	mux.Handle("/", http.FileServer(http.Dir("frontend")))

	// Apply middlewares
	var h http.Handler = mux
	for _, mw := range middlewares {
		h = mw(h)
	}
	return h
}

var middlewares = []middleware.Middleware{
	middleware.RequestLogger(os.Stdout),
	middleware.SetResponseHeader("Cache-Control", "no-cache"),
}
