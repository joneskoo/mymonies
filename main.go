package main

import (
	"flag"
	"log"
	"net"
	"net/http"
	"os"

	"github.com/joneskoo/mymonies/database"
	"github.com/joneskoo/mymonies/handlers"
)

func main() {
	postgres := flag.String("postgres", "database=mymonies", "PostgreSQL connection string")
	flag.Parse()

	log.SetPrefix("[mymonies] ")
	log.SetFlags(log.Lshortfile)

	log.Println("Connecting to database…")
	db, err := database.New(*postgres)
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
	h := handlers.New(db)
	log.Fatal(http.ListenAndServe(laddr, h))
}
