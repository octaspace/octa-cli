package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/octaspace/octa/internal/api"
	"github.com/octaspace/octa/internal/config"
	"github.com/octaspace/octa/internal/ui"
	"github.com/spf13/cobra"
)

var sessionsCmd = &cobra.Command{
	Use:   "sessions",
	Short: "List active sessions",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		client := api.NewClient(cfg.APIKey)

		format, _ := cmd.Flags().GetString("output")
		if format == "json" {
			raw, err := client.ListSessionsRaw()
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			var buf interface{}
			json.Unmarshal(raw, &buf)
			pretty, _ := json.MarshalIndent(buf, "", "  ")
			fmt.Println(string(pretty))
			return nil
		}

		sessions, err := client.ListSessions()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		if len(sessions) == 0 {
			fmt.Println("No active sessions.")
			return nil
		}

		return ui.RenderSessionsTable(sessions)
	},
}

var sessionsStopCmd = &cobra.Command{
	Use:   "stop <uuid>",
	Short: "Stop a session by full or partial UUID",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		input := args[0]

		cfg, err := config.Load()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		client := api.NewClient(cfg.APIKey)

		sessions, err := client.ListSessions()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		var matched []api.Session
		for _, s := range sessions {
			if s.UUID == input || strings.HasPrefix(s.UUID, input) {
				matched = append(matched, s)
			}
		}

		switch len(matched) {
		case 0:
			fmt.Fprintf(os.Stderr, "no session found matching %q\n", input)
			os.Exit(1)
		case 1:
			uuid := matched[0].UUID
			if err := client.StopSession(uuid); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			fmt.Printf("Session %s stopped.\n", uuid)
		default:
			fmt.Fprintf(os.Stderr, "ambiguous UUID %q matches %d sessions:\n", input, len(matched))
			for _, s := range matched {
				fmt.Fprintf(os.Stderr, "  %s\n", s.UUID)
			}
			os.Exit(1)
		}

		return nil
	},
}

func init() {
	sessionsCmd.Flags().StringP("output", "o", "table", "Output format: table or json")
	sessionsCmd.AddCommand(sessionsStopCmd)
}
