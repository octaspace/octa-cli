package ui

import (
	"fmt"
	"math/big"
	"regexp"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	octaspace "github.com/octaspace/go-sdk"
)

// FormatOCTA converts a Wei value to a human-readable OCTA string with the
// given number of fractional digits. A nil value is treated as zero.
func FormatOCTA(wei *big.Int, prec int) string {
	if wei == nil {
		wei = new(big.Int)
	}
	octa := new(big.Float).Quo(new(big.Float).SetInt(wei), new(big.Float).SetFloat64(1e18))
	return fmt.Sprintf("%.*f OCTA", prec, octa)
}

var (
	styleIdle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#cc1b99")).Bold(true)
	styleBusy    = lipgloss.NewStyle().Foreground(lipgloss.Color("#471288")).Bold(true)
	styleOffline = lipgloss.NewStyle().Foreground(lipgloss.Color("#670057")).Bold(true)

	headerStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#cc1b99"))
	borderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#471288"))
)

// RenderNodesTable prints the nodes as a static table.
func RenderNodesTable(nodes []octaspace.Node) error {
	headers := []string{"ID", "State", "CPU", "GPU", "RAM", "Disk", "Location"}

	rows := make([][]string, 0, len(nodes))
	states := make([]string, 0, len(nodes))
	for _, n := range nodes {
		rows = append(rows, nodeToRow(n))
		states = append(states, n.State)
	}

	t := table.New().
		Border(lipgloss.NormalBorder()).
		BorderStyle(borderStyle).
		Headers(headers...).
		StyleFunc(func(row, col int) lipgloss.Style {
			if row == 0 {
				return headerStyle
			}
			// col 1 is State — color by value
			if col == 1 && row-1 < len(states) {
				switch states[row-1] {
				case "idle":
					return styleIdle
				case "busy":
					return styleBusy
				default:
					return styleOffline
				}
			}
			if row%2 == 0 {
				return lipgloss.NewStyle().Foreground(lipgloss.Color("#ffffff"))
			}
			return lipgloss.NewStyle().Foreground(lipgloss.Color("#b088c8"))
		}).
		Rows(rows...)

	fmt.Println(t)
	return nil
}

func nodeToRow(n octaspace.Node) []string {
	// State — plain text, styled via StyleFunc
	state := n.State
	if state == "" {
		state = "offline"
	}

	// Location
	location := fmt.Sprintf("%s, %s", n.Location.City, n.Location.Country)

	// CPU
	cpu := fmt.Sprintf("%dc %s", n.System.CPUCores, cleanModel(n.System.CPUModelName))

	// GPU
	gpu := "-"
	if len(n.System.GPUs) > 0 {
		g := n.System.GPUs[0]
		memGB := (g.MemTotalMB + 512) / 1024
		if len(n.System.GPUs) > 1 {
			gpu = fmt.Sprintf("%dx %s %dGB", len(n.System.GPUs), cleanModel(g.Model), memGB)
		} else {
			gpu = fmt.Sprintf("%s %dGB", cleanModel(g.Model), memGB)
		}
	}

	// RAM
	ramGB := n.System.Memory.Size / 1024 / 1024 / 1024
	ramFreeGB := n.System.Memory.Free / 1024 / 1024 / 1024
	ram := fmt.Sprintf("%d/%d GB", ramFreeGB, ramGB)

	// Disk
	diskGB := n.System.Disk.Size / 1024 / 1024 / 1024
	diskFreeGB := n.System.Disk.Free / 1024 / 1024 / 1024
	disk := fmt.Sprintf("%d/%d GB", diskFreeGB, diskGB)

	return []string{fmt.Sprintf("%d", n.ID), state, cpu, gpu, ram, disk, location}
}

// RenderComputeTable prints available machines for rent as a static table.
func RenderComputeTable(machines []octaspace.MachineRental) error {
	headers := []string{"Node ID", "CPU", "GPU", "RAM", "Disk", "Price/hr", "Location"}

	rows := make([][]string, 0, len(machines))
	for _, m := range machines {
		rows = append(rows, machineToRow(m))
	}

	t := table.New().
		Border(lipgloss.NormalBorder()).
		BorderStyle(borderStyle).
		Headers(headers...).
		StyleFunc(func(row, col int) lipgloss.Style {
			if row == 0 {
				return headerStyle
			}
			if row%2 == 0 {
				return lipgloss.NewStyle().Foreground(lipgloss.Color("#ffffff"))
			}
			return lipgloss.NewStyle().Foreground(lipgloss.Color("#b088c8"))
		}).
		Rows(rows...)

	fmt.Println(t)
	return nil
}

func machineToRow(m octaspace.MachineRental) []string {
	// Location
	location := fmt.Sprintf("%s, %s", m.City, m.Country)

	// CPU
	cpu := fmt.Sprintf("%dc %s", m.CPUCores, cleanModel(m.CPUModelName))

	// GPU
	gpu := "-"
	if len(m.GPUs) > 0 {
		g := m.GPUs[0]
		memGB := (g.MemTotalMB + 512) / 1024
		if len(m.GPUs) > 1 {
			gpu = fmt.Sprintf("%dx %s %dGB", len(m.GPUs), cleanModel(g.Model), memGB)
		} else {
			gpu = fmt.Sprintf("%s %dGB", cleanModel(g.Model), memGB)
		}
	}

	// RAM
	ramGB := m.TotalMemory / 1024 / 1024 / 1024
	ram := fmt.Sprintf("%d GB", ramGB)

	// Disk
	diskGB := m.FreeDisk / 1024 / 1024 / 1024
	disk := fmt.Sprintf("%d GB", diskGB)

	// Price: values are in cents with 5 decimal places (divide by 100000 to get USD)
	price := fmt.Sprintf("B:$%.4f S:$%.4f T:$%.4f",
		float64(m.BaseUSD)/10000,
		float64(m.StorageUSD)/10000,
		float64(m.TrafficUSD)/10000,
	)

	return []string{fmt.Sprintf("%d", m.NodeID), cpu, gpu, ram, disk, price, location}
}

// RenderAppsTable prints available applications as a static table.
func RenderAppsTable(apps []octaspace.App) error {
	headers := []string{"UUID", "Name", "Image"}

	rows := make([][]string, 0, len(apps))
	for _, a := range apps {
		rows = append(rows, []string{a.UUID, a.Name, a.Image})
	}

	t := table.New().
		Border(lipgloss.NormalBorder()).
		BorderStyle(borderStyle).
		Headers(headers...).
		StyleFunc(func(row, col int) lipgloss.Style {
			if row == 0 {
				return headerStyle
			}
			if row%2 == 0 {
				return lipgloss.NewStyle().Foreground(lipgloss.Color("#ffffff"))
			}
			return lipgloss.NewStyle().Foreground(lipgloss.Color("#b088c8"))
		}).
		Rows(rows...)

	fmt.Println(t)
	return nil
}

// RenderSessionsTable prints sessions as a static table.
func RenderSessionsTable(sessions []octaspace.Session) error {
	headers := []string{"UUID", "App", "Node", "Status", "Duration", "Charged", "Disk Size", "SSH Direct", "SSH Proxy", "Web Services"}

	rows := make([][]string, 0, len(sessions))
	for _, s := range sessions {
		rows = append(rows, sessionToRow(s))
	}

	t := table.New().
		Border(lipgloss.NormalBorder()).
		BorderStyle(borderStyle).
		Headers(headers...).
		StyleFunc(func(row, col int) lipgloss.Style {
			if row == 0 {
				return headerStyle
			}
			if row%2 == 0 {
				return lipgloss.NewStyle().Foreground(lipgloss.Color("#ffffff"))
			}
			return lipgloss.NewStyle().Foreground(lipgloss.Color("#b088c8"))
		}).
		Rows(rows...)

	fmt.Println(t)
	return nil
}

func sessionToRow(s octaspace.Session) []string {
	uuid := s.UUID
	if len(uuid) > 8 {
		uuid = uuid[:8]
	}

	status := s.Progress
	if s.IsReady {
		status = "ready"
	}

	dur := s.Duration.Int64()
	h := dur / 3600
	m := (dur % 3600) / 60
	duration := fmt.Sprintf("%dh %02dm", h, m)

	charged := FormatOCTA(s.ChargeAmount.Int, 6)

	sshDirect := "-"
	if s.SSHDirect.Host != "" {
		sshDirect = fmt.Sprintf("%s:%d", s.SSHDirect.Host, s.SSHDirect.Port)
	}

	sshProxy := "-"
	if s.SSHProxy.Host != "" {
		sshProxy = fmt.Sprintf("%s:%d", s.SSHProxy.Host, s.SSHProxy.Port)
	}

	webServices := "-"
	if len(s.URLs) > 0 {
		parts := make([]string, 0, len(s.URLs))
		for name, url := range s.URLs {
			parts = append(parts, fmt.Sprintf("%s: %s", name, url))
		}
		webServices = strings.Join(parts, "\n")
	}

	diskSize := fmt.Sprintf("%d", s.NodeHW.Disk)

	return []string{uuid, s.AppName, fmt.Sprintf("%d", s.NodeID), status, duration, charged, diskSize, sshDirect, sshProxy, webServices}
}

// RenderVPNRelaysTable prints available VPN relays as a static table.
func RenderVPNRelaysTable(relays []octaspace.VPNRelay) error {
	headers := []string{"Node ID", "Location", "Price/GB", "Download", "Upload", "Residential"}

	rows := make([][]string, 0, len(relays))
	for _, r := range relays {
		location := fmt.Sprintf("%s, %s", r.City, r.Country)
		price := fmt.Sprintf("$%.4f", r.PricePerGB)
		download := formatMbps(r.DownloadMbps())
		upload := formatMbps(r.UploadMbps())
		residential := "No"
		if r.Residential {
			residential = "Yes"
		}
		rows = append(rows, []string{fmt.Sprintf("%d", r.NodeID), location, price, download, upload, residential})
	}

	t := table.New().
		Border(lipgloss.NormalBorder()).
		BorderStyle(borderStyle).
		Headers(headers...).
		StyleFunc(func(row, col int) lipgloss.Style {
			if row == 0 {
				return headerStyle
			}
			if row%2 == 0 {
				return lipgloss.NewStyle().Foreground(lipgloss.Color("#ffffff"))
			}
			return lipgloss.NewStyle().Foreground(lipgloss.Color("#b088c8"))
		}).
		Rows(rows...)

	fmt.Println(t)
	return nil
}

// formatMbps formats a Mbit/s value into a human-readable string. The SDK's
// DownloadMbps/UploadMbps helpers already normalize the raw API scale.
func formatMbps(mbps float64) string {
	switch {
	case mbps >= 1_000:
		return fmt.Sprintf("%.1f Gbps", mbps/1_000)
	case mbps >= 1:
		return fmt.Sprintf("%.1f Mbps", mbps)
	case mbps > 0:
		return fmt.Sprintf("%.0f Kbps", mbps*1_000)
	default:
		return "0 bps"
	}
}

var coresSuffixRe = regexp.MustCompile(` \d+-Core$`)

// cleanModel removes brand prefixes and redundant suffixes from CPU/GPU model names.
func cleanModel(model string) string {
	for _, prefix := range []string{"AMD ", "Intel ", "NVIDIA ", "GeForce "} {
		model = strings.TrimPrefix(model, prefix)
	}
	for _, suffix := range []string{" Processor", " CPU"} {
		model = strings.TrimSuffix(model, suffix)
	}
	model = coresSuffixRe.ReplaceAllString(model, "")
	return model
}
