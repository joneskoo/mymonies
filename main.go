// Command mymonies is a personal finance organizer web service.
package main

import (
	"flag"
	"log"
	"net"
	"net/http"
	"os"

	"github.com/gorilla/handlers"
	"github.com/joneskoo/mymonies/api"
	"github.com/joneskoo/mymonies/database"
	"github.com/joneskoo/mymonies/handler"
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
	mux := http.NewServeMux()
	mux.Handle("/api/", http.StripPrefix("/api", api.New(db)))
	// mux.Handle("/old/", http.StripPrefix("/old", handler.New(db)))
	mux.Handle("/static/", http.StripPrefix("/static", http.FileServer(http.Dir("static"))))
	mux.Handle("/", handler.New(db))

	h := handlers.LoggingHandler(os.Stdout, mux)
	log.Fatal(http.ListenAndServe(laddr, h))
}
