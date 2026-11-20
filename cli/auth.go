package cli

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	octaspace "github.com/octaspace/go-sdk"
	"github.com/octaspace/octa/internal/client"
	"github.com/octaspace/octa/internal/config"
	"github.com/spf13/cobra"
)

var authCmd = &cobra.Command{
	Use:   "auth <key>",
	Short: "Verify and save an API key to the config file",
	Long: `Verify an API key and save it to the config file.

Create the key in OctaSpace under Profile → Credentials: it starts with
"octa_ak_" and is shown only once. The key is checked by reading the account
balance.`,
	Args: exactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		token := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(args[0]), "Bearer "))
		if !config.IsAPIKey(token) {
			return fmt.Errorf("not an API key: keys start with %q - create one in OctaSpace under Profile → Credentials", config.APIKeyPrefix)
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

var authStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show the config file, API host and API key in use",
	Long: `Show the config file, the API host and where it comes from ($OCTA_API_URL,
the config file or the default), and the saved API key, masked. A saved key
is checked by reading the account balance. Exits with an error when no
working key is configured.`,
	Args: noArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		path, err := config.Path()
		if err != nil {
			return err
		}
		cfg, err := config.LoadOptional()
		if err != nil {
			return err
		}
		_, statErr := os.Stat(path)
		url, source := cfg.BaseURLSource()

		st := authStatus{
			ConfigFile:   path,
			ConfigExists: statErr == nil,
			APIURL:       url,
			APIURLSource: source,
			APIKey:       maskKey(cfg.APIKey),
		}
		if cfg.APIKey == "" {
			st.KeyStatus = "missing"
		} else {
			_, err := client.New(cfg).Accounts.Balance(cmd.Context())
			switch {
			case err == nil:
				st.KeyStatus = "valid"
			case octaspace.IsAuthError(err):
				st.KeyStatus = "invalid"
			default:
				st.KeyStatus = "unknown"
				st.Error = client.Friendly(err).Error()
			}
		}

		format, _ := cmd.Flags().GetString("output")
		if format == "json" {
			if err := printJSON(st); err != nil {
				return err
			}
		} else {
			printAuthStatus(st)
		}

		switch st.KeyStatus {
		case "missing":
			return errors.New("no API key: run 'octa auth <key>'")
		case "invalid":
			return errors.New("the saved API key is rejected by " + st.APIURL + ": run 'octa auth <key>' with a key from Profile → Credentials")
		case "unknown":
			return errors.New("could not check the API key: " + st.Error)
		}
		return nil
	},
}

// authStatus is the output of 'auth status'.
type authStatus struct {
	ConfigFile   string `json:"config_file"`
	ConfigExists bool   `json:"config_exists"`
	APIURL       string `json:"api_url"`
	APIURLSource string `json:"api_url_source"`
	APIKey       string `json:"api_key"`    // masked; "" when not set
	KeyStatus    string `json:"key_status"` // valid, invalid, missing or unknown
	Error        string `json:"error,omitempty"`
}

func printAuthStatus(st authStatus) {
	label := lipgloss.NewStyle().Bold(true)
	row := func(k, v string) {
		fmt.Printf("%s  %s\n", label.Render(fmt.Sprintf("%-12s", k)), v)
	}

	file := st.ConfigFile
	if !st.ConfigExists {
		file += " (not created)"
	}
	row("Config file", file)
	row("API host", st.APIURL+" ("+st.APIURLSource+")")
	row("API key", orDash(st.APIKey))
	status := map[string]string{
		"valid":   "valid",
		"invalid": "rejected by the API",
		"missing": "not set",
		"unknown": "not checked",
	}[st.KeyStatus]
	row("Key status", status)
}

// maskKey keeps the prefix and four more characters of an API key, enough to
// tell keys apart without revealing them. Anything else is masked whole.
func maskKey(key string) string {
	const visible = len(config.APIKeyPrefix) + 4
	if key == "" {
		return ""
	}
	if !config.IsAPIKey(key) || len(key) <= visible {
		return strings.Repeat("*", len(key))
	}
	return key[:visible] + "…"
}

func init() {
	authCmd.Flags().StringP("output", "o", "table", "Output format: table or json")
	authStatusCmd.Flags().StringP("output", "o", "table", "Output format: table or json")
	authCmd.AddCommand(authStatusCmd)
}
