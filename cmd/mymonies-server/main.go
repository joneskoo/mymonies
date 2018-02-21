package main

import (
	"flag"
	"log"
	"net"
	"net/http"
	"os"

	"github.com/joneskoo/mymonies/pkg/database"
	"github.com/joneskoo/mymonies/pkg/mymoniesserver"
	"github.com/joneskoo/mymonies/pkg/rpc/mymonies"
	// "github.com/joneskoo/mymonies/pkg/twirp-serverhook-prometheus"
)

func main() {
	postgres := flag.String("postgres", "database=mymonies", "PostgreSQL connection string")
	flag.Parse()

	log.SetPrefix("[mymonies] ")
	log.SetFlags(log.Lshortfile)

	log.Println("Connecting to database…")
	db, err := database.Connect(*postgres)
	if err != nil {
		log.Fatal(err)
	}
	if err := db.CreateTables(); err != nil {
		log.Fatal(err)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8000"
	}
	laddr := net.JoinHostPort("127.0.0.1", port)
	log.Println("Listening on http://" + laddr)
	// mux.Handle("/api/", http.StripPrefix("/api", api.New(db)))
	// // mux.Handle("/old/", http.StripPrefix("/old", handler.New(db)))
	// mux.Handle("/static/", http.StripPrefix("/static", http.FileServer(http.Dir("static"))))
	// mux.Handle("/", handler.New(db))

	// h := handlers.LoggingHandler(os.Stdout, mux)
	// log.Fatal(http.ListenAndServe(laddr, h))

	// Initialize twirp RPC handler with prometheus metrics
	server := &mymoniesserver.Server{DB: db}
	// hooks := prometheus.NewServerHooks(nil)
	twirpHandler := mymonies.NewMymoniesServer(server, nil)

	mux := http.NewServeMux()
	mux.Handle(mymonies.MymoniesPathPrefix, twirpHandler)
	// mux.Handle("/metrics", promhttp.Handler())
	mux.Handle("/", http.FileServer(http.Dir("frontend")))

	err = http.ListenAndServe(laddr, noCache(mux))
	log.Fatal(err)
}

func noCache(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-cache")
		h.ServeHTTP(w, r)
	})
}
