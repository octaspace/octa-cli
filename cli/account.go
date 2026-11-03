package cli

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/octaspace/octa/internal/client"
	"github.com/octaspace/octa/internal/config"
	"github.com/octaspace/octa/internal/ui"
	"github.com/spf13/cobra"
)

// The API has no account profile endpoint: the balance is all it reports
// about the account, so 'account' and 'account balance' show the same thing.
var accountCmd = &cobra.Command{
	Use:   "account",
	Short: "Show the account balance",
	Args:  noArgs,
	RunE:  runAccountBalance,
}

var accountBalanceCmd = &cobra.Command{
	Use:   "balance",
	Short: "Show the account balance",
	Args:  noArgs,
	RunE:  runAccountBalance,
}

func runAccountBalance(cmd *cobra.Command, _ []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	b, err := client.New(cfg).Accounts.Balance(cmd.Context())
	if err != nil {
		return client.Friendly(err)
	}

	format, _ := cmd.Flags().GetString("output")
	if format == "json" {
		return printJSON(b)
	}

	fmt.Println(lipgloss.NewStyle().Bold(true).Render(ui.FormatBalance(int64(b.Balance), b.Decimals, b.Currency)))
	return nil
}

func init() {
	accountCmd.Flags().StringP("output", "o", "table", "Output format: table or json")
	accountBalanceCmd.Flags().StringP("output", "o", "table", "Output format: table or json")
	accountCmd.AddCommand(accountBalanceCmd)
}
