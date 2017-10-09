package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/joneskoo/mymonies/database"
	"github.com/joneskoo/mymonies/datasource/nordea"
)

func main() {
	postgres := flag.String("postgres", "database=mymonies", "PostgreSQL connection string")
	flag.Parse()

	log.SetPrefix("[mymonies] ")
	log.SetFlags(log.Lshortfile)

	if flag.NArg() != 1 {
		flag.Usage()
	}
	tsvFile := flag.Arg(0)

	file, err := nordea.FromFile(tsvFile)
	if err != nil {
		log.Fatal(err)
	}
	for _, r := range file.Transactions() {
		r.Tag = classify(*r)
	}

	log.Println("Connecting to database…")
	db, err := database.New(*postgres)
	if err != nil {
		log.Fatal(err)
	}
	if err := db.CreateTables(); err != nil {
		log.Fatal(err)
	}

	err = db.AddImport(database.Import{
		Account:  file.Account(),
		Filename: file.FileName(),
		Records:  file.Transactions(),
	})
	if err != nil {
		log.Fatal(err)
	}

	accounts, err := db.ListAccounts()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Have accounts in database:", accounts)

}

type pattern map[string][]string

var classPatterns = loadClassPatterns()

func loadClassPatterns() (patterns map[string][]string) {
	f, err := os.Open("patterns.json")
	if err != nil {
		panic(err)
	}
	decoder := json.NewDecoder(f)
	if err := decoder.Decode(&patterns); err != nil {
		panic(err)
	}
	f.Close()
	return
}

func classify(r database.Record) string {
	fields := []string{r.PayeePayer, r.Message, r.Reference, r.Account}
	for class, patterns := range classPatterns {
		for _, f := range fields {
			for _, p := range patterns {
				if strings.EqualFold(f, p) {
					return class
				}
			}
		}
	}
	return ""
}
