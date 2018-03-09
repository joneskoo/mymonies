package cmd

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/joneskoo/mymonies/pkg/datasource"
	"github.com/joneskoo/mymonies/pkg/datasource/nordea/pdf"
	"github.com/joneskoo/mymonies/pkg/datasource/nordea/tsv"
	"github.com/joneskoo/mymonies/pkg/rpc/mymonies"
	"github.com/spf13/cobra"
)

var serverAddress string

// importCmd represents the import command
var importCmd = &cobra.Command{
	Use:   "import",
	Short: "Import transaction records into mymonies",
	Long: `The command import reads transactions from different formats and submits them
	to a mymonies server.`,
	Args: requiredFilesWithTypes(".txt", ".pdf"),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		client := mymonies.NewMymoniesProtobufClient(serverAddress, &http.Client{})
		for _, filename := range args {
			f, err := parseFile(filename)
			if err != nil {
				return fmt.Errorf("%v: %v", filename, err)
			}
			_, err = client.AddImport(ctx, &mymonies.AddImportReq{
				Account:      f.Account(),
				FileName:     f.FileName(),
				Transactions: f.Transactions(),
			})
			if err != nil {
				return fmt.Errorf("%v: %v", filename, err)
			}
			fmt.Println(f.FileName(), len(f.Transactions()), "transactions")
		}
		return nil
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

	importCmd.PersistentFlags().StringVar(&serverAddress, "mymonies", "http://127.0.0.1:8000", "Store imported transactions to mymonies server")
	// importCmd.PersistentFlags().String("json", "", "Output imported data as JSON files into directory")
}
