package cli

import (
	"context"

	"github.com/spf13/cobra"
)

var version = "dev"

var rootCmd = &cobra.Command{
	Use:           "octa",
	Short:         "OctaSpace CLI",
	Long:          "A command-line interface for the OctaSpace API.",
	Version:       version,
	SilenceUsage:  true,
	SilenceErrors: true,
}

// Execute runs the root command with the given context. The context is
// propagated to every command via cmd.Context() so API calls honour
// cancellation (e.g. Ctrl-C). Errors are returned to the caller for a single
// point of reporting.
func Execute(ctx context.Context) error {
	return rootCmd.ExecuteContext(ctx)
}

func init() {
	cobra.EnableCommandSorting = false
	rootCmd.AddCommand(authCmd)
	rootCmd.AddCommand(accountCmd)
	rootCmd.AddCommand(computeCmd)
	rootCmd.AddCommand(vpnCmd)
	rootCmd.AddCommand(sessionsCmd)
	rootCmd.AddCommand(nodesCmd)
	rootCmd.AddCommand(idleJobsCmd)
	rootCmd.AddCommand(networkCmd)
}
