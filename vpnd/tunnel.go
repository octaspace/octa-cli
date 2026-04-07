package vpnd

import (
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"

	"golang.zx2c4.com/wireguard/conn"
	"golang.zx2c4.com/wireguard/device"
	"golang.zx2c4.com/wireguard/tun"
)

const tunName = "octa-vpn"

// Tunnel holds an active WireGuard userspace tunnel.
type Tunnel struct {
	dev            *device.Device
	iface          string
	endpointRoutes []string
}

// Interface returns the OS network interface name for this tunnel.
func (t *Tunnel) Interface() string { return t.iface }

// NewTunnel creates a TUN interface, configures WireGuard, assigns addresses and routes.
func NewTunnel(raw string) (*Tunnel, error) {
	cfg, err := parseWGConfig(raw)
	if err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	tdev, err := tun.CreateTUN(tunName, cfg.MTU)
	if err != nil {
		return nil, fmt.Errorf("create TUN: %w", err)
	}

	ifaceName, err := tdev.Name()
	if err != nil {
		tdev.Close()
		return nil, fmt.Errorf("get interface name: %w", err)
	}

	logger := device.NewLogger(device.LogLevelError, fmt.Sprintf("[%s] ", ifaceName))
	dev := device.NewDevice(tdev, conn.NewDefaultBind(), logger)

	if err := dev.IpcSet(cfg.toIPC()); err != nil {
		dev.Close()
		return nil, fmt.Errorf("configure wireguard: %w", err)
	}

	if err := dev.Up(); err != nil {
		dev.Close()
		return nil, fmt.Errorf("bring up device: %w", err)
	}

	if err := ifaceUp(ifaceName, cfg.MTU); err != nil {
		dev.Close()
		return nil, fmt.Errorf("interface up: %w", err)
	}

	for _, addr := range cfg.Addresses {
		if err := addAddress(ifaceName, addr); err != nil {
			dev.Close()
			return nil, fmt.Errorf("add address %s: %w", addr, err)
		}
	}

	needsDefaultRoute := false
	for _, peer := range cfg.Peers {
		for _, cidr := range peer.AllowedIPs {
			if cidr == "0.0.0.0/0" || cidr == "::/0" {
				needsDefaultRoute = true
				break
			}
		}
	}

	var endpointRoutes []string
	if needsDefaultRoute {
		gw, gwIface, err := defaultGateway()
		if err != nil {
			dev.Close()
			return nil, fmt.Errorf("get default gateway: %w", err)
		}
		for _, peer := range cfg.Peers {
			if peer.Endpoint == "" {
				continue
			}
			host := resolveEndpointHost(peer.Endpoint)
			if err := addHostRoute(host, gw, gwIface); err != nil {
				log.Printf("warning: add endpoint route for %s: %v", host, err)
				continue
			}
			endpointRoutes = append(endpointRoutes, host)
		}
	}

	for _, peer := range cfg.Peers {
		for _, cidr := range peer.AllowedIPs {
			for _, r := range splitDefaultRoute(cidr) {
				if err := addRoute(ifaceName, r); err != nil {
					log.Printf("warning: add route %s: %v", r, err)
				}
			}
		}
	}

	log.Printf("tunnel up: %s", ifaceName)
	return &Tunnel{dev: dev, iface: ifaceName, endpointRoutes: endpointRoutes}, nil
}

// Close tears down the tunnel and cleans up routes.
func (t *Tunnel) Close() {
	rx, tx := t.ifaceStats()

	for _, host := range t.endpointRoutes {
		if err := removeHostRoute(host); err != nil {
			log.Printf("warning: remove endpoint route %s: %v", host, err)
		}
	}
	t.dev.Close()
	log.Printf("tunnel down: %s  rx=%s tx=%s", t.iface, fmtBytes(rx), fmtBytes(tx))
}

// ifaceStats returns total rx/tx bytes across all peers via WireGuard UAPI.
func (t *Tunnel) ifaceStats() (rx, tx uint64) {
	ipc, err := t.dev.IpcGet()
	if err != nil {
		return
	}
	for _, line := range strings.Split(ipc, "\n") {
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		n, err := strconv.ParseUint(strings.TrimSpace(v), 10, 64)
		if err != nil {
			continue
		}
		switch strings.TrimSpace(k) {
		case "rx_bytes":
			rx += n
		case "tx_bytes":
			tx += n
		}
	}
	return
}

func fmtBytes(b uint64) string {
	switch {
	case b >= 1<<30:
		return fmt.Sprintf("%.2f GB", float64(b)/float64(1<<30))
	case b >= 1<<20:
		return fmt.Sprintf("%.2f MB", float64(b)/float64(1<<20))
	case b >= 1<<10:
		return fmt.Sprintf("%.2f KB", float64(b)/float64(1<<10))
	default:
		return fmt.Sprintf("%d B", b)
	}
}

func splitDefaultRoute(cidr string) []string {
	switch cidr {
	case "0.0.0.0/0":
		return []string{"0.0.0.0/1", "128.0.0.0/1"}
	case "::/0":
		return []string{"::/1", "8000::/1"}
	default:
		return []string{cidr}
	}
}

func resolveEndpointHost(endpoint string) string {
	host, _, err := net.SplitHostPort(endpoint)
	if err != nil {
		return endpoint
	}
	if net.ParseIP(host) != nil {
		return host
	}
	addrs, err := net.LookupHost(host)
	if err == nil && len(addrs) > 0 {
		return addrs[0]
	}
	return host
}
