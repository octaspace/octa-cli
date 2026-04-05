package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var outputFormat string

var rootCmd = &cobra.Command{
	Use:   "octa",
	Short: "OctaSpace CLI",
	Long:  "A command-line interface for the OctaSpace API.",
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(authCmd)
	rootCmd.AddCommand(nodesCmd)
}
