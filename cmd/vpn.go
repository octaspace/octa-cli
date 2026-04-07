package cmd

import (
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/mdp/qrterminal/v3"
	"github.com/octaspace/octa/internal/api"
	"github.com/octaspace/octa/internal/config"
	"github.com/octaspace/octa/internal/ui"
	"github.com/spf13/cobra"
)

var vpnCmd = &cobra.Command{
	Use:   "vpn",
	Short: "Manage VPN services",
}

func formatTraffic(bytes int64) string {
	switch {
	case bytes >= 1<<30:
		return fmt.Sprintf("%.2f GB", float64(bytes)/float64(1<<30))
	case bytes >= 1<<20:
		return fmt.Sprintf("%.2f MB", float64(bytes)/float64(1<<20))
	case bytes >= 1<<10:
		return fmt.Sprintf("%.2f KB", float64(bytes)/float64(1<<10))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

var vpnConnectCmd = &cobra.Command{
	Use:   "connect",
	Short: "Create a VPN session using the configured relay node",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		if cfg.VPNRelayNode == 0 {
			fmt.Fprintln(os.Stderr, "no VPN relay node configured, run 'octa vpn relay set <node_id>'")
			os.Exit(1)
		}

		protocol, _ := cmd.Flags().GetString("protocol")
		switch protocol {
		case "wg", "ss", "openvpn":
		default:
			fmt.Fprintf(os.Stderr, "error: invalid protocol %q, must be one of: wg, ss, openvpn\n", protocol)
			os.Exit(1)
		}

		resp, err := api.NewClient(cfg.APIKey).ConnectVPN(cfg.VPNRelayNode, protocol)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		fmt.Printf("Session UUID: %s\n", resp.UUID)
		return nil
	},
}

var vpnStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show VPN config for the active session on the configured relay node",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		if cfg.VPNRelayNode == 0 {
			fmt.Fprintln(os.Stderr, "no VPN relay node configured, run 'octa vpn relay set <node_id>'")
			os.Exit(1)
		}

		client := api.NewClient(cfg.APIKey)
		sessions, err := client.ListSessions()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		var activeUUID string
		for _, s := range sessions {
			if int(s.NodeID) == cfg.VPNRelayNode {
				activeUUID = s.UUID
				break
			}
		}

		if activeUUID == "" {
			fmt.Fprintf(os.Stderr, "no active session found for node %d\n", cfg.VPNRelayNode)
			os.Exit(1)
		}

		format, _ := cmd.Flags().GetString("output")
		if format == "json" {
			raw, err := client.GetSessionInfoRaw(activeUUID)
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

		info, err := client.GetSessionInfo(activeUUID)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		showQR, _ := cmd.Flags().GetBool("qr")
		showConfig, _ := cmd.Flags().GetBool("config")

		if showQR {
			qrterminal.GenerateHalfBlock(info.VPNConfig, qrterminal.L, os.Stdout)
			return nil
		}
		if showConfig {
			fmt.Println(info.VPNConfig)
			return nil
		}

		label := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#471288"))
		value := lipgloss.NewStyle().Foreground(lipgloss.Color("#cc1b99"))
		row := func(k, v string) {
			fmt.Printf("%s  %s\n", label.Render(fmt.Sprintf("%-10s", k)), value.Render(v))
		}

		fmt.Println()
		row("Node ID", fmt.Sprintf("%d", cfg.VPNRelayNode))
		row("Country", cfg.VPNRelayCountry)
		row("City", cfg.VPNRelayCity)
		row("Upload", formatTraffic(info.TX))
		row("Download", formatTraffic(info.RX))
		octa := new(big.Float).Quo(new(big.Float).SetInt(info.ChargeAmount.Int), new(big.Float).SetFloat64(1e18))
		row("Charged", fmt.Sprintf("%.10f OCTA", octa))
		fmt.Println()
		return nil
	},
}

var vpnRelayCmd = &cobra.Command{
	Use:   "relay",
	Short: "Manage VPN relays",
}

var vpnRelayListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available VPN relay nodes",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		client := api.NewClient(cfg.APIKey)

		format, _ := cmd.Flags().GetString("output")
		if format == "json" {
			raw, err := client.ListVPNRelaysRaw()
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

		relays, err := client.ListVPNRelays()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		if len(relays) == 0 {
			fmt.Println("No VPN relays available.")
			return nil
		}

		return ui.RenderVPNRelaysTable(relays)
	},
}

var vpnRelaySearchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search VPN relay nodes by country or city",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		query := strings.ToLower(args[0])

		relays, err := api.NewClient(cfg.APIKey).ListVPNRelays()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		var filtered []api.VPNRelay
		for _, r := range relays {
			if strings.Contains(strings.ToLower(r.Country), query) || strings.Contains(strings.ToLower(r.City), query) {
				filtered = append(filtered, r)
			}
		}

		if len(filtered) == 0 {
			fmt.Println("No VPN relays found.")
			return nil
		}

		return ui.RenderVPNRelaysTable(filtered)
	},
}

var vpnRelaySetCmd = &cobra.Command{
	Use:   "set <node_id>",
	Short: "Set the VPN relay node",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var nodeID int
		if _, err := fmt.Sscan(args[0], &nodeID); err != nil || nodeID <= 0 {
			fmt.Fprintln(os.Stderr, "error: node_id must be a positive integer")
			os.Exit(1)
		}

		cfg, err := config.Load()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		relays, err := api.NewClient(cfg.APIKey).ListVPNRelays()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		var relay *api.VPNRelay
		for i := range relays {
			if relays[i].NodeID == nodeID {
				relay = &relays[i]
				break
			}
		}
		if relay == nil {
			fmt.Fprintf(os.Stderr, "error: node %d not found\n", nodeID)
			os.Exit(1)
		}

		cfg.VPNRelayNode = nodeID
		cfg.VPNRelayCountry = relay.Country
		cfg.VPNRelayCity = relay.City
		if err := config.Save(cfg); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		fmt.Printf("VPN relay node set to %d (%s, %s).\n", nodeID, relay.City, relay.Country)
		return nil
	},
}

var vpnRelayGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Show the configured VPN relay node",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		if cfg.VPNRelayNode == 0 {
			fmt.Fprintln(os.Stderr, "no VPN relay node configured, run 'octa vpn relay set <node_id>'")
			os.Exit(1)
		}

		fmt.Printf("Node ID: %d\nCity:    %s\nCountry: %s\n", cfg.VPNRelayNode, cfg.VPNRelayCity, cfg.VPNRelayCountry)
		return nil
	},
}

func init() {
	vpnRelayListCmd.Flags().StringP("output", "o", "table", "Output format: table or json")
	vpnRelayCmd.AddCommand(vpnRelayListCmd)
	vpnRelayCmd.AddCommand(vpnRelaySearchCmd)
	vpnRelayCmd.AddCommand(vpnRelaySetCmd)
	vpnRelayCmd.AddCommand(vpnRelayGetCmd)
	vpnCmd.AddCommand(vpnRelayCmd)
	vpnConnectCmd.Flags().String("protocol", "wg", "VPN protocol: wg, ss, openvpn")
	vpnStatusCmd.Flags().StringP("output", "o", "table", "Output format: table or json")
	vpnStatusCmd.Flags().Bool("qr", false, "Display VPN config as QR code")
	vpnStatusCmd.Flags().Bool("config", false, "Display plain VPN config")
	vpnCmd.AddCommand(vpnConnectCmd)
	vpnCmd.AddCommand(vpnStatusCmd)
}
