package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/octaspace/octa/internal/api"
	"github.com/octaspace/octa/internal/config"
	"github.com/octaspace/octa/internal/ui"
	"github.com/spf13/cobra"
)

var vpnCmd = &cobra.Command{
	Use:   "vpn",
	Short: "Manage VPN services",
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
	vpnRelayCmd.AddCommand(vpnRelaySetCmd)
	vpnRelayCmd.AddCommand(vpnRelayGetCmd)
	vpnCmd.AddCommand(vpnRelayCmd)
	vpnConnectCmd.Flags().String("protocol", "wg", "VPN protocol: wg, ss, openvpn")
	vpnCmd.AddCommand(vpnConnectCmd)
}
