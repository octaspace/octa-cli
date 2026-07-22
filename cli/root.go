package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

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

// exactArgs returns a cobra.PositionalArgs validator that requires exactly n
// positional arguments. Unlike cobra.ExactArgs, the error names the missing
// or extra arguments and shows the command's usage line, since SilenceUsage
// is enabled globally and the default cobra message ("accepts 1 arg(s),
// received 0") would otherwise reach the user with no context.
func exactArgs(n int) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if len(args) == n {
			return nil
		}
		if len(args) < n {
			return fmt.Errorf("missing required argument(s): %s\nUsage: %s", missingArgNames(cmd, n, len(args)), cmd.UseLine())
		}
		return fmt.Errorf("too many arguments (expected %d, got %d)\nUsage: %s", n, len(args), cmd.UseLine())
	}
}

// missingArgNames extracts the `<...>` placeholder names from a command's Use
// string, skipping the first `got` of them, and joins the rest for use in an
// error message. Falls back to a generic count if Use has no placeholders.
func missingArgNames(cmd *cobra.Command, want, got int) string {
	var names []string
	rest := cmd.Use
	for {
		start := strings.IndexByte(rest, '<')
		if start == -1 {
			break
		}
		end := strings.IndexByte(rest[start:], '>')
		if end == -1 {
			break
		}
		names = append(names, rest[start:start+end+1])
		rest = rest[start+end+1:]
	}
	if len(names) < want || got >= len(names) {
		return fmt.Sprintf("expected %d, got %d", want, got)
	}
	return strings.Join(names[got:], ", ")
}

// printJSON marshals v as indented JSON and prints it to stdout. Used by
// commands that have no raw-JSON passthrough from the API (e.g. because the
// SDK only exposes a decoded struct) but still support -o json.
func printJSON(v any) error {
	out, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(out))
	return nil
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
