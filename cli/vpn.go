package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/mdp/qrterminal/v3"
	octaspace "github.com/octaspace/go-sdk"
	"github.com/octaspace/octa/internal/client"
	"github.com/octaspace/octa/internal/config"
	"github.com/octaspace/octa/internal/ui"
	"github.com/octaspace/octa/internal/vpnd"
	"github.com/spf13/cobra"
)

const (
	vpnReadyTimeout   = 2 * time.Minute
	vpnPollInterval   = 3 * time.Second
	vpnCleanupTimeout = 15 * time.Second
)

type vpnSessionInfoGetter interface {
	Info(context.Context) (*octaspace.SessionInfo, error)
}

type vpnSessionStopper interface {
	Stop(context.Context, *octaspace.StopParams) error
}

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
			return err
		}

		if cfg.VPNRelayNode == 0 {
			return errors.New("no VPN relay node configured, run 'octa vpn relay set <node_id>'")
		}

		// protocol maps to the API "subkind" field. For v2ray an additional
		// subprotocol (--v2ray-protocol) is required, mirroring Cube.
		protocol, _ := cmd.Flags().GetString("protocol")
		v2rayProtocol, _ := cmd.Flags().GetString("v2ray-protocol")
		switch protocol {
		case "wg", "ss", "openvpn":
		case "v2ray":
			switch v2rayProtocol {
			case "vless", "vmess", "trojan":
			default:
				return fmt.Errorf("--v2ray-protocol is required for v2ray and must be one of: vless, vmess, trojan")
			}
		default:
			return fmt.Errorf("invalid protocol %q, must be one of: wg, ss, openvpn, v2ray", protocol)
		}

		c := client.New(cfg)

		params := &octaspace.VPNCreateParams{
			NodeID:  int64(cfg.VPNRelayNode),
			SubKind: protocol,
		}
		if protocol == "v2ray" {
			params.Protocol = v2rayProtocol
		}

		resp, err := c.Services.VPN.Create(cmd.Context(), params)
		if err != nil {
			return client.Friendly(err)
		}
		if resp.UUID == "" {
			return errors.New("VPN API returned an empty session UUID")
		}

		fmt.Printf("Session UUID: %s\n", resp.UUID)

		if protocol != "wg" {
			return nil
		}
		session := c.Services.Session(resp.UUID)

		// For WireGuard: wait for the session to become ready, then bring up the tunnel.
		fmt.Print("Waiting for session to be ready")
		vpnCfg, err := waitForVPNConfig(cmd.Context(), session, vpnReadyTimeout, vpnPollInterval, func() {
			fmt.Print(".")
		})
		fmt.Println()
		if err != nil {
			return rollbackVPNConnect(session, err)
		}

		wgResp, err := vpnd.Connect(vpnCfg)
		if err != nil {
			return rollbackVPNConnect(session, fmt.Errorf("tunnel error: %w", err))
		}
		if !wgResp.OK {
			return rollbackVPNConnect(session, fmt.Errorf("tunnel error: %s", wgResp.Error))
		}

		cfg.VPNSessionUUID = resp.UUID
		if err := config.Save(cfg); err != nil {
			cause := fmt.Errorf("could not save session UUID: %w", err)
			if disconnectErr := disconnectWireGuardAfterFailure(); disconnectErr != nil {
				cause = errors.Join(cause, disconnectErr)
			}
			return rollbackVPNConnect(session, cause)
		}

		fmt.Printf("Tunnel up: %s\n", wgResp.Interface)
		return nil
	},
}

var vpnDisconnectCmd = &cobra.Command{
	Use:   "disconnect",
	Short: "Tear down the active WireGuard tunnel and stop the VPN session",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		wgResp, err := vpnd.Disconnect()
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: tunnel unavailable: %v\n", err)
		} else if !wgResp.OK && wgResp.Error != "no active tunnel" {
			fmt.Fprintf(os.Stderr, "warning: tunnel error: %s\n", wgResp.Error)
		} else {
			fmt.Println("Tunnel down.")
		}

		if cfg.VPNSessionUUID == "" {
			return nil
		}

		if err := client.New(cfg).Services.Session(cfg.VPNSessionUUID).Stop(cmd.Context(), nil); err != nil {
			if !octaspace.IsNotFound(err) {
				return fmt.Errorf("could not stop session: %w", client.Friendly(err))
			}
		} else {
			fmt.Printf("Session %s stopped.\n", shortSessionUUID(cfg.VPNSessionUUID))
		}

		cfg.VPNSessionUUID = ""
		if err := config.Save(cfg); err != nil {
			return fmt.Errorf("could not update config: %w", err)
		}

		return nil
	},
}

var vpnStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show VPN config for the active session on the configured relay node",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		if cfg.VPNRelayNode == 0 {
			return errors.New("no VPN relay node configured, run 'octa vpn relay set <node_id>'")
		}

		c := client.New(cfg)
		sessions, err := c.Sessions.List(cmd.Context(), nil)
		if err != nil {
			return client.Friendly(err)
		}

		activeUUID := activeVPNSessionUUID(sessions, cfg)

		if activeUUID == "" {
			return fmt.Errorf("no active session found for node %d", cfg.VPNRelayNode)
		}

		format, _ := cmd.Flags().GetString("output")
		if format == "json" {
			out, err := c.Services.Session(activeUUID).InfoRaw(cmd.Context())
			if err != nil {
				return client.Friendly(err)
			}
			fmt.Println(string(out))
			return nil
		}

		info, err := c.Services.Session(activeUUID).Info(cmd.Context())
		if err != nil {
			return client.Friendly(err)
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
		row("Charged", ui.FormatOCTA(info.ChargeAmount.Int, 10))
		fmt.Println()
		return nil
	},
}

func waitForVPNConfig(
	ctx context.Context,
	session vpnSessionInfoGetter,
	timeout time.Duration,
	pollInterval time.Duration,
	onPoll func(),
) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if timeout <= 0 {
		return "", errors.New("VPN readiness timeout must be positive")
	}
	if pollInterval <= 0 {
		return "", errors.New("VPN poll interval must be positive")
	}

	waitCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		info, err := session.Info(waitCtx)
		if err == nil && info != nil && info.VPNConfig != "" {
			return info.VPNConfig, nil
		}
		if waitErr := waitCtx.Err(); waitErr != nil {
			return "", vpnWaitError(ctx, waitErr)
		}

		select {
		case <-waitCtx.Done():
			return "", vpnWaitError(ctx, waitCtx.Err())
		case <-ticker.C:
			if onPoll != nil {
				onPoll()
			}
		}
	}
}

func vpnWaitError(parent context.Context, waitErr error) error {
	if err := parent.Err(); err != nil {
		return err
	}
	if errors.Is(waitErr, context.DeadlineExceeded) {
		return errors.New("timed out waiting for VPN config")
	}
	return waitErr
}

func rollbackVPNConnect(session vpnSessionStopper, cause error) error {
	cleanupCtx, cancel := context.WithTimeout(context.Background(), vpnCleanupTimeout)
	defer cancel()
	if err := session.Stop(cleanupCtx, nil); err != nil && !octaspace.IsNotFound(err) {
		return errors.Join(cause, fmt.Errorf("could not stop created VPN session: %w", client.Friendly(err)))
	}
	return cause
}

func disconnectWireGuardAfterFailure() error {
	resp, err := vpnd.Disconnect()
	if err != nil {
		return fmt.Errorf("could not roll back local tunnel: %w", err)
	}
	if !resp.OK && resp.Error != "no active tunnel" {
		return fmt.Errorf("could not roll back local tunnel: %s", resp.Error)
	}
	return nil
}

func activeVPNSessionUUID(sessions []octaspace.Session, cfg *config.Config) string {
	isVPN := func(session octaspace.Session) bool {
		return session.Service == "" || session.Service == "vpn"
	}
	if cfg.VPNSessionUUID != "" {
		for _, session := range sessions {
			if session.UUID == cfg.VPNSessionUUID && isVPN(session) {
				return session.UUID
			}
		}
	}
	for _, session := range sessions {
		if int(session.NodeID) == cfg.VPNRelayNode && isVPN(session) {
			return session.UUID
		}
	}
	return ""
}

func shortSessionUUID(uuid string) string {
	if len(uuid) > 8 {
		return uuid[:8]
	}
	return uuid
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
			return err
		}

		c := client.New(cfg)
		format, _ := cmd.Flags().GetString("output")
		if format == "json" {
			out, err := c.Services.VPN.ListRaw(cmd.Context(), nil)
			if err != nil {
				return client.Friendly(err)
			}
			fmt.Println(string(out))
			return nil
		}
		relays, err := c.Services.VPN.List(cmd.Context(), nil)
		if err != nil {
			return client.Friendly(err)
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
			return err
		}

		query := strings.ToLower(args[0])

		relays, err := client.New(cfg).Services.VPN.List(cmd.Context(), nil)
		if err != nil {
			return client.Friendly(err)
		}

		residential, _ := cmd.Flags().GetBool("residential")

		var filtered []octaspace.VPNRelay
		for _, r := range relays {
			if strings.Contains(strings.ToLower(r.Country), query) || strings.Contains(strings.ToLower(r.City), query) {
				if residential && !r.Residential {
					continue
				}
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
			return errors.New("node_id must be a positive integer")
		}

		cfg, err := config.Load()
		if err != nil {
			return err
		}

		relays, err := client.New(cfg).Services.VPN.List(cmd.Context(), nil)
		if err != nil {
			return client.Friendly(err)
		}

		var relay *octaspace.VPNRelay
		for i := range relays {
			if int(relays[i].NodeID) == nodeID {
				relay = &relays[i]
				break
			}
		}
		if relay == nil {
			return fmt.Errorf("node %d not found", nodeID)
		}

		cfg.VPNRelayNode = nodeID
		cfg.VPNRelayCountry = relay.Country
		cfg.VPNRelayCity = relay.City
		if err := config.Save(cfg); err != nil {
			return err
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
			return err
		}

		if cfg.VPNRelayNode == 0 {
			return errors.New("no VPN relay node configured, run 'octa vpn relay set <node_id>'")
		}

		fmt.Printf("Node ID: %d\nCity:    %s\nCountry: %s\n", cfg.VPNRelayNode, cfg.VPNRelayCity, cfg.VPNRelayCountry)
		return nil
	},
}

func init() {
	vpnRelayListCmd.Flags().StringP("output", "o", "table", "Output format: table or json")
	vpnRelaySearchCmd.Flags().Bool("residential", false, "Show only residential nodes")
	vpnRelayCmd.AddCommand(vpnRelayListCmd)
	vpnRelayCmd.AddCommand(vpnRelaySearchCmd)
	vpnRelayCmd.AddCommand(vpnRelaySetCmd)
	vpnRelayCmd.AddCommand(vpnRelayGetCmd)
	vpnCmd.AddCommand(vpnRelayCmd)
	vpnConnectCmd.Flags().String("protocol", "wg", "VPN type: wg, ss, openvpn, v2ray")
	vpnConnectCmd.Flags().String("v2ray-protocol", "", "V2Ray subprotocol (required for --protocol v2ray): vless, vmess, trojan")
	vpnStatusCmd.Flags().StringP("output", "o", "table", "Output format: table or json")
	vpnStatusCmd.Flags().Bool("qr", false, "Display VPN config as QR code")
	vpnStatusCmd.Flags().Bool("config", false, "Display plain VPN config")
	vpnCmd.AddCommand(vpnConnectCmd)
	vpnCmd.AddCommand(vpnDisconnectCmd)
	vpnCmd.AddCommand(vpnStatusCmd)
}
