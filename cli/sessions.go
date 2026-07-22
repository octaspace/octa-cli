package cli

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
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
	Args:  exactArgs(1),
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

		format, _ := cmd.Flags().GetString("output")
		if format == "json" {
			return printJSON(map[string]any{"status": "ok", "uuid": session.UUID})
		}
		fmt.Printf("Session %s stopped.\n", session.UUID)
		return nil
	},
}

var sessionsInfoCmd = &cobra.Command{
	Use:   "info <uuid>",
	Short: "Show detailed information about a session",
	Args:  exactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		c := client.New(cfg)
		sessions, err := c.Sessions.List(cmd.Context(), nil)
		if err != nil {
			return client.Friendly(err)
		}

		session, err := client.MatchSession(sessions, args[0])
		if err != nil {
			return err
		}

		proxy := c.Services.Session(session.UUID)
		format, _ := cmd.Flags().GetString("output")
		if format == "json" {
			out, err := proxy.InfoRaw(cmd.Context())
			if err != nil {
				return client.Friendly(err)
			}
			fmt.Println(string(out))
			return nil
		}

		info, err := proxy.Info(cmd.Context())
		if err != nil {
			return client.Friendly(err)
		}

		label := lipgloss.NewStyle().Bold(true)
		value := lipgloss.NewStyle()
		row := func(k, v string) {
			fmt.Printf("%s  %s\n", label.Render(fmt.Sprintf("%-16s", k)), value.Render(v))
		}

		status := info.Progress
		if info.IsReady {
			status = "ready"
		}

		fmt.Println()
		row("UUID", info.UUID)
		row("App", info.AppName)
		row("Service / Kind", fmt.Sprintf("%s / %s", info.Service, info.Kind))
		row("Node", fmt.Sprintf("%d", info.NodeID))
		row("Status", status)
		row("Location", fmt.Sprintf("%s, %s", info.City, info.Country))
		row("Public IP", info.PublicIP)
		row("Container", info.ContainerID.String())
		if info.SSHDirect.Host != "" {
			row("SSH Direct", fmt.Sprintf("%s:%d", info.SSHDirect.Host, info.SSHDirect.Port))
		}
		if info.SSHProxy.Host != "" {
			row("SSH Proxy", fmt.Sprintf("%s:%d", info.SSHProxy.Host, info.SSHProxy.Port))
		}
		row("Traffic", fmt.Sprintf("TX %d / RX %d bytes", info.TX, info.RX))
		row("Charged", ui.FormatOCTA(info.ChargeAmount.Int, 6))
		for name, u := range info.URLs {
			row("URL: "+name, u)
		}
		fmt.Println()
		return nil
	},
}

func init() {
	sessionsCmd.Flags().StringP("output", "o", "table", "Output format: table or json")
	sessionsStopCmd.Flags().StringP("output", "o", "table", "Output format: table or json")
	sessionsInfoCmd.Flags().StringP("output", "o", "table", "Output format: table or json")
	sessionsCmd.AddCommand(sessionsStopCmd)
	sessionsCmd.AddCommand(sessionsInfoCmd)
}
