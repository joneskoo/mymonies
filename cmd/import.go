package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/joneskoo/mymonies/pkg/datasource"
	"github.com/joneskoo/mymonies/pkg/datasource/nordea/pdf"
	"github.com/joneskoo/mymonies/pkg/datasource/nordea/tsv"
	"github.com/spf13/cobra"
)

// importCmd represents the import command
var importCmd = &cobra.Command{
	Use:   "import",
	Short: "Import transaction records into mymonies",
	Long: `The command import reads transactions from different formats and submits them
	to a mymonies server.`,
	Args: requiredFilesWithTypes(".txt", ".pdf"),
	RunE: func(cmd *cobra.Command, args []string) error {
		_, err := readFiles(args)
		if err != nil {
			return err
		}
		return nil
		// ctx := context.Background()
		// client := mymonies.NewMymoniesProtobufClient(*server, &http.Client{})
		// _, err = client.AddImport(ctx, &mymonies.AddImportReq{
		// 	Account:      file.Account(),
		// 	FileName:     file.FileName(),
		// 	Transactions: file.Transactions(),
		// })
		// if err != nil {
		// 	logger.Fatalf("%v: %v", filename, err)
		// }
		// // fmt.Printf("Account: %q File: %q\n", f.Account(), f.FileName())
		// // for _, tx := range f.Transactions() {
		// // 	fmt.Printf("%v %12v %-40v %8.2f %v\n", tx.TransactionDate.Format("02.01.2006"), tx.Transaction, tx.PayeePayer, tx.Amount, tx.Tag)
		// // }
		// // fmt.Println("---------------------------")

	},
}

func requiredFilesWithTypes(types ...string) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return fmt.Errorf("no files to import")
		}
		for _, f := range args {
			_, err := os.Stat(f)
			if err != nil {
				return err
			}
			for _, t := range types {
				if filepath.Ext(f) == t {
					return nil
				}
			}
			return fmt.Errorf("%v: unsupported file extension, only %q are supported", f, types)

		}
		return nil
	}
}

func readFiles(args []string) ([]datasource.File, error) {
	files := make([]datasource.File, 0)
	for _, filename := range args {
		f, err := parseFile(filename)
		if err != nil {
			return nil, fmt.Errorf("%v: %v", filename, err)
		}
		files = append(files, f)
		fmt.Println(filename)
	}
	return files, nil
}

func parseFile(filename string) (datasource.File, error) {
	ext := filepath.Ext(filename)
	switch ext {
	case ".pdf":
		return pdf.FromFile(filename)
	case ".tsv":
		return tsv.FromFile(filename)
	case ".txt":
		return tsv.FromFile(filename)
	default:
		return nil, fmt.Errorf("file type extension %q is not supported", ext)
	}

}

func init() {
	rootCmd.AddCommand(importCmd)

	importCmd.PersistentFlags().String("mymonies", "http://127.0.0.1:8000", "Store imported transactions to mymonies server")
	importCmd.PersistentFlags().String("json", "", "Output imported data as JSON files into directory")
}
