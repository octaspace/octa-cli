package vpnd

import (
	"bufio"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
)

// wgConfig holds a parsed WireGuard config file.
type wgConfig struct {
	PrivateKey string   // hex-encoded
	Addresses  []string // CIDR strings
	DNS        []string
	ListenPort int
	MTU        int
	Peers      []wgPeer
}

type wgPeer struct {
	PublicKey           string // hex-encoded
	PresharedKey        string // hex-encoded, optional
	Endpoint            string
	AllowedIPs          []string
	PersistentKeepalive int
}

func parseWGConfig(raw string) (*wgConfig, error) {
	cfg := &wgConfig{MTU: 1420}
	var currentPeer *wgPeer

	scanner := bufio.NewScanner(strings.NewReader(raw))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if line == "[Interface]" {
			currentPeer = nil
			continue
		}
		if line == "[Peer]" {
			cfg.Peers = append(cfg.Peers, wgPeer{})
			currentPeer = &cfg.Peers[len(cfg.Peers)-1]
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		k := strings.TrimSpace(parts[0])
		v := strings.TrimSpace(parts[1])

		if currentPeer == nil {
			switch k {
			case "PrivateKey":
				h, err := b64ToHex(v)
				if err != nil {
					return nil, fmt.Errorf("PrivateKey: %w", err)
				}
				cfg.PrivateKey = h
			case "Address":
				cfg.Addresses = append(cfg.Addresses, splitCSV(v)...)
			case "DNS":
				cfg.DNS = append(cfg.DNS, splitCSV(v)...)
			case "ListenPort":
				p, err := strconv.Atoi(v)
				if err != nil {
					return nil, fmt.Errorf("ListenPort: %w", err)
				}
				cfg.ListenPort = p
			case "MTU":
				m, err := strconv.Atoi(v)
				if err != nil {
					return nil, fmt.Errorf("MTU: %w", err)
				}
				cfg.MTU = m
			}
		} else {
			switch k {
			case "PublicKey":
				h, err := b64ToHex(v)
				if err != nil {
					return nil, fmt.Errorf("peer PublicKey: %w", err)
				}
				currentPeer.PublicKey = h
			case "PresharedKey":
				h, err := b64ToHex(v)
				if err != nil {
					return nil, fmt.Errorf("PresharedKey: %w", err)
				}
				currentPeer.PresharedKey = h
			case "AllowedIPs":
				currentPeer.AllowedIPs = append(currentPeer.AllowedIPs, splitCSV(v)...)
			case "Endpoint":
				currentPeer.Endpoint = v
			case "PersistentKeepalive":
				ka, err := strconv.Atoi(v)
				if err != nil {
					return nil, fmt.Errorf("PersistentKeepalive: %w", err)
				}
				currentPeer.PersistentKeepalive = ka
			}
		}
	}

	if cfg.PrivateKey == "" {
		return nil, fmt.Errorf("missing PrivateKey in [Interface]")
	}

	return cfg, nil
}

func (cfg *wgConfig) toIPC() string {
	var sb strings.Builder

	fmt.Fprintf(&sb, "private_key=%s\n", cfg.PrivateKey)
	if cfg.ListenPort > 0 {
		fmt.Fprintf(&sb, "listen_port=%d\n", cfg.ListenPort)
	}

	for _, peer := range cfg.Peers {
		fmt.Fprintf(&sb, "public_key=%s\n", peer.PublicKey)
		if peer.PresharedKey != "" {
			fmt.Fprintf(&sb, "preshared_key=%s\n", peer.PresharedKey)
		}
		if peer.Endpoint != "" {
			fmt.Fprintf(&sb, "endpoint=%s\n", peer.Endpoint)
		}
		if peer.PersistentKeepalive > 0 {
			fmt.Fprintf(&sb, "persistent_keepalive_interval=%d\n", peer.PersistentKeepalive)
		}
		for _, ip := range peer.AllowedIPs {
			fmt.Fprintf(&sb, "allowed_ip=%s\n", ip)
		}
	}

	return sb.String()
}

func b64ToHex(s string) (string, error) {
	b, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func splitCSV(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
