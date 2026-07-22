package cli

import (
	"encoding/json"
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/octaspace/octa/internal/client"
	"github.com/octaspace/octa/internal/config"
	"github.com/octaspace/octa/internal/ui"
	"github.com/spf13/cobra"
)

var accountCmd = &cobra.Command{
	Use:   "account",
	Short: "Show account information",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		c := client.New(cfg)
		format, _ := cmd.Flags().GetString("output")
		if format == "json" {
			out, err := c.Accounts.ProfileRaw(cmd.Context())
			if err != nil {
				return client.Friendly(err)
			}
			fmt.Println(string(out))
			return nil
		}
		account, err := c.Accounts.Profile(cmd.Context())
		if err != nil {
			return client.Friendly(err)
		}

		label := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#471288"))
		value := lipgloss.NewStyle().Foreground(lipgloss.Color("#cc1b99"))

		row := func(k, v string) {
			fmt.Printf("%s  %s\n", label.Render(fmt.Sprintf("%-14s", k)), value.Render(v))
		}

		fmt.Println()
		row("UID", fmt.Sprintf("%d", account.UID))
		row("Email", account.Email)
		row("Wallet", account.Wallet)
		row("Balance", ui.FormatOCTA(account.Balance.Int, 4))
		fmt.Println()

		return nil
	},
}

var accountBalanceCmd = &cobra.Command{
	Use:   "balance",
	Short: "Show account balance",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		c := client.New(cfg)
		balanceWei, err := c.Accounts.Balance(cmd.Context())
		if err != nil {
			return client.Friendly(err)
		}

		format, _ := cmd.Flags().GetString("output")
		if format == "json" {
			out, _ := json.MarshalIndent(map[string]string{"balance": balanceWei.String()}, "", "  ")
			fmt.Println(string(out))
			return nil
		}

		style := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#cc1b99"))
		fmt.Println(style.Render(ui.FormatOCTA(balanceWei, 4)))
		return nil
	},
}

func init() {
	accountCmd.Flags().StringP("output", "o", "table", "Output format: table or json")
	accountBalanceCmd.Flags().StringP("output", "o", "table", "Output format: table or json")
	accountCmd.AddCommand(accountBalanceCmd)
}
