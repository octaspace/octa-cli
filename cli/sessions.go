package cli

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	octaspace "github.com/octaspace/go-sdk"
	"github.com/octaspace/octa/internal/client"
	"github.com/octaspace/octa/internal/config"
	"github.com/octaspace/octa/internal/ui"
	"github.com/spf13/cobra"
)

var sessionsCmd = &cobra.Command{
	Use:   "sessions",
	Short: "List running sessions",
	Long: `List the account's running sessions, oldest first.
The API keeps no history: a stopped session is no longer listed.`,
	Args: noArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		c := client.New(cfg)
		format, _ := cmd.Flags().GetString("output")
		if format == "json" {
			out, err := c.Sessions.ListRaw(cmd.Context())
			if err != nil {
				return client.Friendly(err)
			}
			fmt.Println(string(out))
			return nil
		}
		sessions, err := c.Sessions.List(cmd.Context())
		if err != nil {
			return client.Friendly(err)
		}

		if len(sessions) == 0 {
			fmt.Println("No running sessions.")
			return nil
		}

		return ui.RenderSessionsTable(sessions)
	},
}

var sessionsStopCmd = &cobra.Command{
	Use:   "stop <uuid>... | --all",
	Short: "Stop sessions by full or partial UUID, or all of them",
	Long: `Stop one or more sessions by full or partial UUID, or every running session
with --all. All UUIDs are resolved before anything is stopped, so a typo stops
nothing. Stopping is asynchronous: a session may still be listed for a few
seconds. A session stopped before its minimum rent is charged base and storage
for the whole minimum.

With several sessions, -o json prints [{"uuid", "status", "reason"}]
(status 0 = stopping, 1 = failed).`,
	Example: `  octa sessions stop 3f2a
  octa sessions stop 3f2a 9c41 e7d0
  octa sessions stop --all`,
	Args: func(cmd *cobra.Command, args []string) error {
		all, _ := cmd.Flags().GetBool("all")
		switch {
		case all && len(args) > 0:
			return fmt.Errorf("--all takes no UUIDs\n\nUsage: %s", cmd.UseLine())
		case !all && len(args) == 0:
			return fmt.Errorf("missing <uuid> (or pass --all)\n\nUsage: %s", cmd.UseLine())
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		c := client.New(cfg)
		running, err := c.Sessions.List(cmd.Context())
		if err != nil {
			return client.Friendly(err)
		}
		targets, err := stopTargets(running, args)
		if err != nil {
			return err
		}

		format, _ := cmd.Flags().GetString("output")
		jsonOut := format == "json"
		if len(targets) == 0 {
			if jsonOut {
				return printJSON([]stopOutcome{})
			}
			fmt.Println("No running sessions.")
			return nil
		}

		outcomes := make([]stopOutcome, 0, len(targets))
		failed := 0
		for _, session := range targets {
			o := stopOutcome{UUID: session.UUID}
			if _, err := c.Sessions.Stop(cmd.Context(), session.UUID); err != nil {
				o.Status, o.Reason = 1, client.Friendly(err).Error()
				failed++
			} else if warning := minRentWarning(session); warning != "" {
				fmt.Fprintln(os.Stderr, warning) // stderr, so it does not break -o json
			}
			outcomes = append(outcomes, o)
		}

		single := len(args) == 1
		switch {
		case jsonOut && single && failed == 0:
			if err := printJSON(map[string]any{"status": "ok", "uuid": outcomes[0].UUID}); err != nil {
				return err
			}
		case jsonOut && !single:
			if err := printJSON(outcomes); err != nil {
				return err
			}
		case !jsonOut:
			for _, o := range outcomes {
				if o.Status == 0 {
					fmt.Printf("Session %s is stopping.\n", o.UUID)
				} else if !single {
					fmt.Printf("Session %s: %s\n", o.UUID, o.Reason)
				}
			}
		}

		switch {
		case failed == 0:
			return nil
		case single:
			return errors.New(outcomes[0].Reason)
		default:
			return fmt.Errorf("%d of %d stops failed", failed, len(outcomes))
		}
	},
}

// stopOutcome is the result of stopping one session.
type stopOutcome struct {
	UUID   string `json:"uuid"`
	Status int    `json:"status"` // 0 - stopping, 1 - failed
	Reason string `json:"reason,omitempty"`
}

// stopTargets resolves the sessions to stop: every running session when
// queries is empty (--all), otherwise one per full or partial UUID, without
// duplicates. Any query that does not resolve fails the whole call.
func stopTargets(running []octaspace.Session, queries []string) ([]octaspace.Session, error) {
	if len(queries) == 0 {
		return running, nil
	}
	targets := make([]octaspace.Session, 0, len(queries))
	seen := make(map[string]bool, len(queries))
	for _, q := range queries {
		s, err := client.MatchSession(running, q)
		if err != nil {
			return nil, err
		}
		if !seen[s.UUID] {
			seen[s.UUID] = true
			targets = append(targets, s)
		}
	}
	return targets, nil
}

// minRentWarning explains the minimum rent charge of a session stopped
// before its minimum rent; "" when it does not apply.
func minRentWarning(s octaspace.Session) string {
	minimum := int64(s.MinRentMinutes) * 60
	if minimum == 0 || s.Duration >= minimum {
		return ""
	}
	return fmt.Sprintf("Note: the session ran %s of its %d-minute minimum rent; base and storage are charged for the whole minimum.",
		ui.FormatDuration(s.Duration), s.MinRentMinutes)
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
		session, err := findSession(cmd.Context(), c, args[0])
		if err != nil {
			return err
		}

		format, _ := cmd.Flags().GetString("output")
		if format == "json" {
			out, err := c.Sessions.GetRaw(cmd.Context(), session.UUID)
			if err != nil {
				return client.Friendly(err)
			}
			fmt.Println(string(out))
			return nil
		}

		info, err := c.Sessions.Get(cmd.Context(), session.UUID)
		if err != nil {
			return client.Friendly(err)
		}
		printSessionInfo(*info)
		return nil
	},
}

func printSessionInfo(s octaspace.Session) {
	label := lipgloss.NewStyle().Bold(true)
	row := func(k, v string) {
		if v == "" {
			return
		}
		fmt.Printf("%s  %s\n", label.Render(fmt.Sprintf("%-16s", k)), v)
	}

	fmt.Println()
	row("UUID", s.UUID)
	row("Service", s.Service)
	row("App", s.AppName)
	row("Node", s.NodeNumber)
	row("Status", ui.SessionStatus(s))
	if t := s.StartedTime(); !t.IsZero() {
		row("Started", t.Local().Format("2006-01-02 15:04:05"))
	}
	row("Duration", ui.FormatDuration(s.Duration))
	if s.City != "" || s.Country != "" {
		row("Location", strings.Trim(s.City+", "+s.Country, ", "))
	}
	row("Public IP", s.PublicIP)
	if s.NodeHW != nil {
		row("Hardware", hardwareSummary(*s.NodeHW))
	}

	prices := "base " + ui.FormatUSD(s.BasePrice) + "/h"
	if s.IsCompute() {
		prices += ", storage " + ui.FormatUSD(s.StoragePrice) + "/GB·h, traffic " + ui.FormatUSD(s.TrafficPrice) + "/GB"
	}
	row("Prices", prices)
	if s.MinRentMinutes > 0 {
		row("Min rent", fmt.Sprintf("%d min", s.MinRentMinutes))
	}
	row("Charged", ui.FormatUSD(s.ChargeAmount))

	if s.IsCompute() {
		row("Traffic", fmt.Sprintf("RX %s / TX %s", ui.FormatBytes(s.RX), ui.FormatBytes(s.TX)))
		if a := ui.SSHAddress(s.SSH.Direct); a != "" {
			row("SSH Direct", "ssh "+a)
		}
		if a := ui.SSHAddress(s.SSH.Proxy); a != "" {
			row("SSH Proxy", "ssh "+a)
		}
		row("Web terminal", s.SSH.Web)
		row("File manager", s.FM)
		for _, k := range sortedKeys(s.Ports) {
			row("Port "+k, fmt.Sprintf("%s:%d", orHost(s.PublicIP), s.Ports[k]))
		}
		for _, k := range sortedKeys(s.URLs) {
			row("URL "+k, s.URLs[k])
		}
		for _, k := range sortedKeys(s.Envs) {
			row("Env "+k, s.Envs[k])
		}
	}
	fmt.Println()
}

func hardwareSummary(hw octaspace.SessionNodeHW) string {
	var parts []string
	if hw.CPUCores > 0 {
		cpu := fmt.Sprintf("%d cores", hw.CPUCores)
		if hw.CPU != "" {
			cpu = fmt.Sprintf("%s (%d cores)", hw.CPU, hw.CPUCores)
		}
		parts = append(parts, cpu)
	}
	if hw.TotalMemory > 0 {
		parts = append(parts, ui.FormatBytes(hw.TotalMemory)+" RAM")
	}
	if hw.Disk > 0 {
		disk := fmt.Sprintf("%d GB disk", hw.Disk)
		if hw.DiskType != "" {
			disk += " " + strings.ToUpper(hw.DiskType)
		}
		parts = append(parts, disk)
	}
	for i := 0; i < len(hw.GPUs); {
		g, n := hw.GPUs[i], 1
		for i+n < len(hw.GPUs) && hw.GPUs[i+n] == g {
			n++
		}
		gpu := fmt.Sprintf("%s %dGB", g.Model, (g.MemTotalMB+512)/1024)
		if n > 1 {
			gpu = fmt.Sprintf("%dx %s", n, gpu)
		}
		parts = append(parts, gpu)
		i += n
	}
	if net := ui.FormatNet(hw.NetDownMbps, hw.NetUpMbps); net != "-" {
		parts = append(parts, "net "+net)
	}
	return strings.Join(parts, ", ")
}

func orHost(ip string) string {
	if ip == "" {
		return "<node>"
	}
	return ip
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func init() {
	sessionsCmd.Flags().StringP("output", "o", "table", "Output format: table or json")
	sessionsStopCmd.Flags().StringP("output", "o", "table", "Output format: table or json")
	sessionsStopCmd.Flags().Bool("all", false, "Stop every running session")
	sessionsInfoCmd.Flags().StringP("output", "o", "table", "Output format: table or json")
	sessionsCmd.AddCommand(sessionsStopCmd)
	sessionsCmd.AddCommand(sessionsInfoCmd)
}
