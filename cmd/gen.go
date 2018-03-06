package cmd

import (
	"github.com/spf13/cobra"
)

// genCmd represents the gen command
var genCmd = &cobra.Command{
	Use:   "gen",
	Short: "A collection of several useful generators.",
}

func init() {
	rootCmd.AddCommand(genCmd)
}
