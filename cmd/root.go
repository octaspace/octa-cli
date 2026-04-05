package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var version = "dev"

var outputFormat string

var rootCmd = &cobra.Command{
	Use:     "octa",
	Short:   "OctaSpace CLI",
	Long:    "A command-line interface for the OctaSpace API.",
	Version: version,
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.EnableCommandSorting = false
	rootCmd.AddCommand(authCmd)
	rootCmd.AddCommand(accountCmd)
	rootCmd.AddCommand(computeCmd)
	rootCmd.AddCommand(vpnCmd)
	rootCmd.AddCommand(sessionsCmd)
	rootCmd.AddCommand(nodesCmd)
}
