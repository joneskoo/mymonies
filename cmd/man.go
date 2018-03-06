package cmd

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

// manCmd represents the man command
var manCmd = &cobra.Command{
	Use:   "man",
	Short: "Generate man page for mymonies",
	RunE: func(cmd *cobra.Command, _ []string) error {
		path, _ := cmd.Flags().GetString("dir")
		os.MkdirAll(path, 0755)
		return doc.GenManTreeFromOpts(rootCmd, doc.GenManTreeOptions{
			Header: &doc.GenManHeader{
				Title:   "MYMONIES",
				Section: "3",
			},
			Path: path,
		})
	},
}

func init() {
	genCmd.AddCommand(manCmd)

	manCmd.Flags().String("dir", "man/", "Target directory for man pages")
}
