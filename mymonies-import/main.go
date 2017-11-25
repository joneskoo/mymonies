// Command mymonies-import imports Nordea TSV files into database.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/joneskoo/mymonies/database"
	"github.com/joneskoo/mymonies/datasource"
	"github.com/joneskoo/mymonies/datasource/nordea/pdf"
	"github.com/joneskoo/mymonies/datasource/nordea/tsv"
)

func main() {
	postgres := flag.String("postgres", "", "PostgreSQL connection string, e.g. database=mymonies")
	flag.Parse()

	log.SetPrefix("[mymonies] ")
	log.SetFlags(log.Lshortfile)

	if flag.NArg() == 0 {
		flag.Usage()
		os.Exit(2)
	}

	var db *database.Postgres
	if *postgres != "" {
		log.Println("Connecting to database…")
		var err error
		db, err = database.Connect(*postgres)
		if err != nil {
			log.Fatal(err)
		}
		if err := db.CreateTables(); err != nil {
			log.Fatal(err)
		}
	} else {
		log.Println("No database URL set, data will not be saved")
	}

	for _, filename := range flag.Args() {
		err := importFile(filename, db)
		if err != nil {
			log.Fatal(err)
		}
	}
	log.Println("All done")
}

func importFile(filename string, db *database.Postgres) error {
	ext := filepath.Ext(filename)
	var f datasource.File
	var err error

	switch ext {
	case ".pdf":
		f, err = pdf.FromFile(filename)
	case ".tsv":
		fallthrough
	case ".txt":
		f, err = tsv.FromFile(filename)
	default:
		return fmt.Errorf("unsupported file type extension %q", ext)
	}

	if err != nil {
		return fmt.Errorf("parsing %v: %v", filename, err)
	}

	// fmt.Printf("Account: %q File: %q\n", f.Account(), f.FileName())
	// for _, tx := range f.Transactions() {
	// 	fmt.Printf("%v %12v %-40v %8.2f %v\n", tx.TransactionDate.Format("02.01.2006"), tx.Transaction, tx.PayeePayer, tx.Amount, tx.Tag)
	// }
	// fmt.Println("---------------------------")

	if db == nil || len(f.Transactions()) == 0 {
		return nil
	}
	log.Println("Saving to database")
	err = db.AddImport(database.ImportTransactions{
		Import: database.Import{
			Account:  f.Account(),
			Filename: f.FileName(),
		},
		Transactions: f.Transactions(),
	})
	if err != nil {
		return err
	}
	return nil
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

func classify(r database.Transaction) string {
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
