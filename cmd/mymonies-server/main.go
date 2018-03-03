package main

import (
	"flag"
	"log"
	"net"
	"net/http"
	"os"

	"github.com/joneskoo/mymonies/pkg/mymoniesserver"
)

func main() {
	conn := flag.String("postgres", "database=mymonies", "PostgreSQL connection string")
	listen := flag.String("listen", defaultListen(), "HTTP server listen address")
	flag.Parse()

	logger := log.New(os.Stdout, "[mymonies] ", log.Lshortfile)
	logger.Println("Listening on http://" + *listen)

	h, err := mymoniesserver.New(*conn, logger)
	if err != nil {
		logger.Fatal(err)
	}
	logger.Fatal(http.ListenAndServe(*listen, h))
}

func defaultListen() string {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8000"
	}
	return net.JoinHostPort("127.0.0.1", port)

}
