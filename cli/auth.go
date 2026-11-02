package cli

import (
	"fmt"
	"strings"

	"github.com/octaspace/octa/internal/client"
	"github.com/octaspace/octa/internal/config"
	"github.com/spf13/cobra"
)

var authCmd = &cobra.Command{
	Use:   "auth <key>",
	Short: "Verify and save an API key to the config file",
	Long: `Verify an API key and save it to the config file.

Create the key in OctaSpace under Profile → Credentials: it starts with
"octa_ak_" and is shown only once. Tokens of the legacy API are rejected
without being sent. The key is checked by reading the account balance.`,
	Args: exactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		token := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(args[0]), "Bearer "))
		if !config.IsAPIKey(token) {
			return fmt.Errorf("not an API key: keys start with %q — create one in OctaSpace under Profile → Credentials (legacy API tokens are not accepted)", config.APIKeyPrefix)
		}

		cfg, err := config.LoadOptional()
		if err != nil {
			return err
		}
		cfg.APIKey = token

		if _, err := client.New(cfg).Accounts.Balance(cmd.Context()); err != nil {
			return client.Friendly(err)
		}

		if err := config.Save(cfg); err != nil {
			return err
		}

		format, _ := cmd.Flags().GetString("output")
		if format == "json" {
			return printJSON(map[string]any{"status": "ok"})
		}
		fmt.Println("API key saved successfully.")
		return nil
	},
}

func init() {
	authCmd.Flags().StringP("output", "o", "table", "Output format: table or json")
}
