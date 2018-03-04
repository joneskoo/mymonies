// Command mymonies-import imports transaction records into mymonies.
package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"

	"github.com/joneskoo/mymonies/pkg/datasource"
	"github.com/joneskoo/mymonies/pkg/rpc/mymonies"
)

func main() {
	server := flag.String("url", "http://127.0.0.1:8000", "Mymonies server address")
	flag.Parse()

	if flag.NArg() == 0 {
		flag.Usage()
		os.Exit(2)
	}

	ctx := context.Background()
	logger := log.New(os.Stdout, "", log.Lshortfile)
	client := mymonies.NewMymoniesProtobufClient(*server, &http.Client{})

	for _, filename := range flag.Args() {
		req, err := datasource.ParseFile(filename)
		if err != nil {
			logger.Fatalf("parsing %v: %v", filename, err)
		}
		_, err = client.AddImport(ctx, req)
		if err != nil {
			logger.Fatal(err)
		}
		// fmt.Printf("Account: %q File: %q\n", f.Account(), f.FileName())
		// for _, tx := range f.Transactions() {
		// 	fmt.Printf("%v %12v %-40v %8.2f %v\n", tx.TransactionDate.Format("02.01.2006"), tx.Transaction, tx.PayeePayer, tx.Amount, tx.Tag)
		// }
		// fmt.Println("---------------------------")
	}
	logger.Println("All done")
}
