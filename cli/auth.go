package cli

import (
	"fmt"
	"os"

	"github.com/octaspace/octa/internal/api"
	"github.com/octaspace/octa/internal/config"
	"github.com/spf13/cobra"
)

var authCmd = &cobra.Command{
	Use:   "auth <token>",
	Short: "Verify and save API token to config file",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		token := args[0]

		client := api.NewClient(token)
		if err := client.ValidateToken(); err != nil {
			fmt.Fprintln(os.Stderr, "Invalid token:", err)
			os.Exit(1)
		}

		if err := config.Save(&config.Config{APIKey: token}); err != nil {
			return err
		}

		fmt.Println("Token saved successfully.")
		return nil
	},
}
