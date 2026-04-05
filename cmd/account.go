package cmd

import (
	"encoding/json"
	"fmt"
	"math/big"
	"os"

	"github.com/charmbracelet/lipgloss"
	"github.com/octaspace/octa/internal/api"
	"github.com/octaspace/octa/internal/config"
	"github.com/spf13/cobra"
)

var accountCmd = &cobra.Command{
	Use:   "account",
	Short: "Show account information",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		client := api.NewClient(cfg.APIKey)
		account, err := client.GetAccount()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		format, _ := cmd.Flags().GetString("output")
		if format == "json" {
			out, _ := json.MarshalIndent(account, "", "  ")
			fmt.Println(string(out))
			return nil
		}

		// Convert balance from Wei to OCTA
		decimals := new(big.Float).SetInt(account.Balance.Int)
		divisor := new(big.Float).SetFloat64(1e18)
		octa := new(big.Float).Quo(decimals, divisor)

		label := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#471288"))
		value := lipgloss.NewStyle().Foreground(lipgloss.Color("#cc1b99"))

		row := func(k, v string) {
			fmt.Printf("%s  %s\n", label.Render(fmt.Sprintf("%-14s", k)), value.Render(v))
		}

		fmt.Println()
		row("UID", fmt.Sprintf("%d", account.UID))
		row("Email", account.Email)
		row("Wallet", account.Wallet)
		row("Balance", fmt.Sprintf("%.4f OCTA", octa))
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
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		client := api.NewClient(cfg.APIKey)
		balanceWei, err := client.GetBalance()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		format, _ := cmd.Flags().GetString("output")
		if format == "json" {
			out, _ := json.MarshalIndent(map[string]string{"balance": balanceWei.String()}, "", "  ")
			fmt.Println(string(out))
			return nil
		}

		octa := new(big.Float).Quo(new(big.Float).SetInt(balanceWei), new(big.Float).SetFloat64(1e18))

		style := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#cc1b99"))
		fmt.Println(style.Render(fmt.Sprintf("%.4f OCTA", octa)))
		return nil
	},
}

func init() {
	accountCmd.Flags().StringP("output", "o", "table", "Output format: table or json")
	accountBalanceCmd.Flags().StringP("output", "o", "table", "Output format: table or json")
	accountCmd.AddCommand(accountBalanceCmd)
}
