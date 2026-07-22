package cli

import (
	"fmt"

	"github.com/octaspace/octa/internal/client"
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
			return err
		}

		c := client.New(cfg)
		format, _ := cmd.Flags().GetString("output")
		if format == "json" {
			out, err := c.Sessions.ListRaw(cmd.Context(), nil)
			if err != nil {
				return client.Friendly(err)
			}
			fmt.Println(string(out))
			return nil
		}
		sessions, err := c.Sessions.List(cmd.Context(), nil)
		if err != nil {
			return client.Friendly(err)
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
			return err
		}

		c := client.New(cfg)
		sessions, err := c.Sessions.List(cmd.Context(), nil)
		if err != nil {
			return client.Friendly(err)
		}

		session, err := client.MatchSession(sessions, input)
		if err != nil {
			return err
		}

		if err := c.Services.Session(session.UUID).Stop(cmd.Context(), nil); err != nil {
			return client.Friendly(err)
		}
		fmt.Printf("Session %s stopped.\n", session.UUID)
		return nil
	},
}

func init() {
	sessionsCmd.Flags().StringP("output", "o", "table", "Output format: table or json")
	sessionsCmd.AddCommand(sessionsStopCmd)
}
