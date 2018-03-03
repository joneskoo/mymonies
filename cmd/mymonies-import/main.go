// Command mymonies-import imports Nordea TSV files into database.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/joneskoo/mymonies/pkg/rpc/mymonies"

	"github.com/joneskoo/mymonies/pkg/datasource/nordea/pdf"
	"github.com/joneskoo/mymonies/pkg/datasource/nordea/tsv"
	"github.com/joneskoo/mymonies/pkg/mymoniesserver/database"
)

func main() {
	server := flag.String("url", "http://127.0.0.1:8000", "Mymonies server address")
	flag.Parse()

	if flag.NArg() == 0 {
		flag.Usage()
		os.Exit(2)
	}

	client := mymonies.NewMymoniesProtobufClient(*server, &http.Client{})

	for _, filename := range flag.Args() {
		err := importFile(filename, client)
		if err != nil {
			log.Fatal(err)
		}
	}
	log.Println("All done")
}

func importFile(filename string, client mymonies.Mymonies) error {
	ext := filepath.Ext(filename)
	var f interface {
		Transactions() []interface{}
	}
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

	if client == nil || len(f.Transactions()) == 0 {
		return nil
	}
	log.Println("Saving to database")
	err = client.AddImport(f.Account(), f.FileName(), f.Transactions())
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
