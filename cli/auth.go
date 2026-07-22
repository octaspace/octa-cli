package cli

import (
	"fmt"

	"github.com/octaspace/octa/internal/client"
	"github.com/octaspace/octa/internal/config"
	"github.com/spf13/cobra"
)

var authCmd = &cobra.Command{
	Use:   "auth <token>",
	Short: "Verify and save API token to config file",
	Args:  exactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		token := args[0]

		c := client.New(&config.Config{APIKey: token})
		if _, err := c.Accounts.Profile(cmd.Context()); err != nil {
			return client.Friendly(err)
		}

		if err := config.Save(&config.Config{APIKey: token}); err != nil {
			return err
		}

		format, _ := cmd.Flags().GetString("output")
		if format == "json" {
			return printJSON(map[string]any{"status": "ok"})
		}
		fmt.Println("Token saved successfully.")
		return nil
	},
}

func init() {
	authCmd.Flags().StringP("output", "o", "table", "Output format: table or json")
}
