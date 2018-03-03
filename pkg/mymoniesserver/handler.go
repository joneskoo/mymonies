package mymoniesserver

import (
	"net/http"
	"os"

	"github.com/joneskoo/mymonies/pkg/middleware"
	"github.com/joneskoo/mymonies/pkg/rpc/mymonies"
	// "github.com/joneskoo/mymonies/pkg/twirp-serverhook-prometheus"
)

var middlewares = []middleware.Middleware{
	middleware.RequestLogger(os.Stdout),
	middleware.SetResponseHeader("Cache-Control", "no-cache"),
}

func (s *Server) handler() http.Handler {
	mux := http.NewServeMux()

	// Twirp RPC handler with prometheus metrics
	// hooks := prometheus.NewServerHooks(nil)
	twirpHandler := mymonies.NewMymoniesServer(s, nil)
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
