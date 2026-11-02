package ui

import (
	"fmt"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	octaspace "github.com/octaspace/go-sdk"
)

var headerStyle = lipgloss.NewStyle().Bold(true)

// FormatUSD formats a micro-USD amount as dollars, trimming trailing zeros
// but keeping at least two decimal places: 100000 → "$0.10", 100 →
// "$0.0001", 2175268 → "$2.175268", -1500000 → "-$1.50".
func FormatUSD(m octaspace.MicroUSD) string {
	return formatDecimal(int64(m), 6, "$")
}

// FormatBalance formats an account balance reported as an integer amount with
// the given number of decimals, e.g. (2175268, 6, "USD") → "$2.175268".
func FormatBalance(amount int64, decimals int, currency string) string {
	if decimals < 0 || decimals > 18 {
		return fmt.Sprintf("%d (×10^-%d %s)", amount, decimals, currency)
	}
	if currency == "USD" || currency == "" {
		return formatDecimal(amount, decimals, "$")
	}
	return formatDecimal(amount, decimals, "") + " " + currency
}

// formatDecimal prints amount / 10^decimals with a sign and prefix, trimming
// trailing zeros down to two decimal places.
func formatDecimal(amount int64, decimals int, prefix string) string {
	sign := ""
	// Work on the absolute value as uint64: negating math.MinInt64 overflows.
	abs := uint64(amount)
	if amount < 0 {
		sign = "-"
		abs = uint64(-(amount + 1)) + 1
	}
	digits := strconv.FormatUint(abs, 10)
	if len(digits) <= decimals {
		digits = strings.Repeat("0", decimals-len(digits)+1) + digits
	}
	whole, frac := digits[:len(digits)-decimals], digits[len(digits)-decimals:]
	frac = strings.TrimRight(frac, "0")
	for len(frac) < 2 && decimals >= 2 {
		frac += "0"
	}
	if frac == "" {
		return sign + prefix + whole
	}
	return sign + prefix + whole + "." + frac
}

// FormatDuration formats seconds as "3h 07m" (or "2d 03h" from a day on).
func FormatDuration(seconds int64) string {
	if seconds < 0 {
		seconds = 0
	}
	d, h, m := seconds/86400, (seconds%86400)/3600, (seconds%3600)/60
	if d > 0 {
		return fmt.Sprintf("%dd %02dh", d, h)
	}
	return fmt.Sprintf("%dh %02dm", h, m)
}

// FormatBytes formats a byte count in binary units ("1.5 GB", "512 MB").
func FormatBytes(b int64) string {
	switch {
	case b >= 1<<40:
		return fmt.Sprintf("%.1f TB", float64(b)/float64(int64(1)<<40))
	case b >= 1<<30:
		return fmt.Sprintf("%.1f GB", float64(b)/float64(1<<30))
	case b >= 1<<20:
		return fmt.Sprintf("%.0f MB", float64(b)/float64(1<<20))
	case b >= 1<<10:
		return fmt.Sprintf("%.0f KB", float64(b)/float64(1<<10))
	default:
		return fmt.Sprintf("%d B", b)
	}
}

// FormatNet formats a download/upload pair in Mbit/s, as the API reports
// them; 0 means not measured.
func FormatNet(down, up int) string {
	if down == 0 && up == 0 {
		return "-"
	}
	return fmt.Sprintf("%s/%s", formatMbps(down), formatMbps(up))
}

// formatMbps formats whole Mbit/s: "-" when unknown, "1.2 Gbps" from 1000.
func formatMbps(mbps int) string {
	switch {
	case mbps <= 0:
		return "-"
	case mbps >= 1000:
		return fmt.Sprintf("%.1f Gbps", float64(mbps)/1000)
	default:
		return fmt.Sprintf("%d Mbps", mbps)
	}
}

// FormatMinRent formats a minimum rent in minutes; "-" when there is none.
func FormatMinRent(minutes int) string {
	if minutes <= 0 {
		return "-"
	}
	return fmt.Sprintf("%dm", minutes)
}

// SessionStatus describes a session's status: "ready", or the configuration
// step while starting, with the image pull progress when known.
func SessionStatus(s octaspace.Session) string {
	if s.IsReady() {
		return octaspace.SessionReady
	}
	status := s.Status
	if s.Progress != "" {
		status = s.Progress
	}
	if s.PullPercent != nil {
		status = fmt.Sprintf("%s (%d%%)", status, *s.PullPercent)
	}
	if status == "" {
		status = octaspace.SessionStarting
	}
	return status
}

// SSHAddress formats an SSH endpoint as "user@host -p port"; "" when nil.
func SSHAddress(e *octaspace.SSHEndpoint) string {
	if e == nil || e.Host == "" || e.Port == 0 {
		return ""
	}
	user := e.User
	if user == "" {
		user = "root"
	}
	return fmt.Sprintf("%s@%s -p %d", user, e.Host, e.Port)
}

func newTable(headers []string, rows [][]string) *table.Table {
	return table.New().
		Border(lipgloss.NormalBorder()).
		Headers(headers...).
		StyleFunc(func(row, col int) lipgloss.Style {
			if row == 0 {
				return headerStyle
			}
			return lipgloss.NewStyle()
		}).
		Rows(rows...)
}

func newDetailTable(rows [][]string) *table.Table {
	return table.New().
		Border(lipgloss.NormalBorder()).
		StyleFunc(func(row, col int) lipgloss.Style {
			if col == 0 {
				return headerStyle
			}
			return lipgloss.NewStyle()
		}).
		Rows(rows...)
}

func location(city, country string) string {
	switch {
	case city != "" && country != "":
		return city + ", " + country
	case country != "":
		return country
	case city != "":
		return city
	default:
		return "-"
	}
}

// gpuSummary formats a GPU list as "2x RTX 3090 24GB" from its first model.
func gpuSummary(count int, model string, memMB int) string {
	if count == 0 {
		return "-"
	}
	label := cleanModel(model)
	if memMB > 0 {
		label = fmt.Sprintf("%s %dGB", label, (memMB+512)/1024)
	}
	if count > 1 {
		return fmt.Sprintf("%dx %s", count, label)
	}
	return label
}

// RenderComputeTable prints the machines available for rent.
func RenderComputeTable(machines []octaspace.ComputeMachine) error {
	headers := []string{"Node", "CPU", "GPU", "RAM", "Disk", "Net ↓/↑", "Price/hr", "Storage, Traffic", "Min rent", "Location"}
	rows := make([][]string, 0, len(machines))
	for _, m := range machines {
		rows = append(rows, computeMachineRow(m))
	}
	fmt.Println(newTable(headers, rows))
	return nil
}

func computeMachineRow(m octaspace.ComputeMachine) []string {
	gpu := "-"
	if len(m.GPUs) > 0 {
		gpu = gpuSummary(len(m.GPUs), m.GPUs[0].Model, m.GPUs[0].MemTotalMB)
	}
	disk := FormatBytes(m.Disk)
	if m.DiskType != "" {
		disk += " " + strings.ToUpper(m.DiskType)
	}
	extra := fmt.Sprintf("%s/GB·h, %s/GB", FormatUSD(m.StoragePrice), FormatUSD(m.TrafficPrice))
	return []string{
		m.NodeNumber,
		fmt.Sprintf("%dc %s", m.CPUCores, cleanModel(m.CPUModelName)),
		gpu,
		FormatBytes(m.Memory),
		disk,
		FormatNet(m.NetDownMbps, m.NetUpMbps),
		FormatUSD(m.BasePrice),
		extra,
		FormatMinRent(m.MinRentMinutes),
		location(m.City, m.Country),
	}
}

// RenderRenderTable prints the nodes available for render.
func RenderRenderTable(machines []octaspace.RenderMachine) error {
	headers := []string{"Node", "GPU", "Blender", "CPU", "RAM", "Net ↓/↑", "Price/hr", "Min rent", "Location"}
	rows := make([][]string, 0, len(machines))
	for _, m := range machines {
		gpu := "-"
		if len(m.GPUs) > 0 {
			gpu = gpuSummary(len(m.GPUs), m.GPUs[0].Model, m.GPUs[0].MemTotalMB)
		}
		rows = append(rows, []string{
			m.NodeNumber,
			gpu,
			strconv.FormatInt(m.BlenderScore, 10),
			fmt.Sprintf("%dc %s", m.CPUCores, cleanModel(m.CPUModelName)),
			FormatBytes(m.Memory),
			FormatNet(m.NetDownMbps, m.NetUpMbps),
			FormatUSD(m.BasePrice),
			FormatMinRent(m.MinRentMinutes),
			location(m.City, m.Country),
		})
	}
	fmt.Println(newTable(headers, rows))
	return nil
}

// RenderSessionsTable prints the running sessions.
func RenderSessionsTable(sessions []octaspace.Session) error {
	headers := []string{"UUID", "Service", "App", "Node", "Status", "Duration", "Charged", "SSH", "Web Services"}
	rows := make([][]string, 0, len(sessions))
	for _, s := range sessions {
		rows = append(rows, sessionToRow(s))
	}
	fmt.Println(newTable(headers, rows))
	return nil
}

func sessionToRow(s octaspace.Session) []string {
	uuid := s.UUID
	if len(uuid) > 8 {
		uuid = uuid[:8]
	}

	ssh := SSHAddress(s.SSH.Direct)
	if ssh == "" {
		ssh = SSHAddress(s.SSH.Proxy)
	}
	if ssh == "" {
		ssh = "-"
	}

	return []string{
		uuid,
		orDash(s.Service),
		orDash(s.AppName),
		orDash(s.NodeNumber),
		SessionStatus(s),
		FormatDuration(s.Duration),
		FormatUSD(s.ChargeAmount),
		ssh,
		joinURLs(s.URLs),
	}
}

// joinURLs lists "port: url" pairs by port, one per line; "-" when empty.
func joinURLs(urls map[string]string) string {
	if len(urls) == 0 {
		return "-"
	}
	keys := make([]string, 0, len(urls))
	for k := range urls {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s: %s", k, urls[k]))
	}
	return strings.Join(parts, "\n")
}

// RenderHostingNodesTable prints the account's online nodes.
func RenderHostingNodesTable(nodes []octaspace.HostingNode) error {
	headers := []string{"Number", "Status", "Offered", "CPU", "GPU", "RAM", "Disk", "Base/hr", "Min rent", "Uptime", "Location"}
	rows := make([][]string, 0, len(nodes))
	for _, n := range nodes {
		rows = append(rows, []string{
			n.Number,
			HostingStatus(n),
			OfferedServices(n.Services),
			fmt.Sprintf("%dc %s", n.System.CPUCores, cleanModel(n.System.CPUModelName)),
			hostingGPUSummary(n.System.GPUs),
			usageSummary(n.System.Memory),
			usageSummary(n.System.Disk),
			FormatUSD(n.Prices.Base),
			FormatMinRent(n.Prices.MinRentMinutes),
			uptimePercent(n.UptimePercent),
			location(n.Location.City, n.Location.Country),
		})
	}
	fmt.Println(newTable(headers, rows))
	return nil
}

// RenderHostingNodeDetail prints one node of the account.
func RenderHostingNodeDetail(n octaspace.HostingNode) error {
	verified := "unknown"
	if n.Verified != nil {
		verified = yesNo(*n.Verified)
	}
	uptime := "-"
	if n.Uptime != nil {
		uptime = FormatDuration(*n.Uptime)
	}
	online := "-"
	if !n.OnlineAt.IsZero() {
		online = n.OnlineAt.Local().Format(time.DateTime)
	}
	cpu := fmt.Sprintf("%dc %s", n.System.CPUCores, cleanModel(n.System.CPUModelName))
	if n.System.CPULoadPercent != nil {
		cpu += fmt.Sprintf(" (%.0f%% load)", *n.System.CPULoadPercent)
	}

	rows := [][]string{
		{"Number", n.Number},
		{"Status", HostingStatus(n)},
		{"Maintenance", yesNo(n.Maintenance)},
		{"Verified", verified},
		{"Services", serviceDetail(n.Services)},
		{"Prices", fmt.Sprintf("base %s/h, storage %s/GB·h, traffic %s/GB", FormatUSD(n.Prices.Base), FormatUSD(n.Prices.Storage), FormatUSD(n.Prices.Traffic))},
		{"Min rent", FormatMinRent(n.Prices.MinRentMinutes)},
		{"IP", orDash(n.IP)},
		{"Agent (OSN)", orDash(n.OSN)},
		{"Online since", online},
		{"Uptime", uptime + ", " + uptimePercent(n.UptimePercent) + " over 30 days"},
		{"Location", location(n.Location.City, n.Location.Country)},
		{"CPU", cpu},
		{"OS", orDash(n.System.OSVersion)},
		{"RAM", usageDetail(n.System.Memory)},
		{"Disk", usageDetail(n.System.Disk)},
	}
	if len(n.System.GPUs) == 0 {
		rows = append(rows, []string{"GPU", "-"})
	}
	for i, g := range n.System.GPUs {
		label := "GPU"
		if len(n.System.GPUs) > 1 {
			label = fmt.Sprintf("GPU %d", i)
		}
		rows = append(rows, []string{label, hostingGPUDetail(g)})
	}

	fmt.Println(newDetailTable(rows))
	return nil
}

// HostingStatus is the node status with the running check, if any.
func HostingStatus(n octaspace.HostingNode) string {
	status := orDash(n.Status)
	if n.Maintenance {
		status += " (maintenance)"
	}
	if n.Activity != nil && n.Activity.Step != "" {
		status += ": " + n.Activity.Step
	}
	return status
}

// OfferedServices lists the services a node is offered for; "-" for none.
func OfferedServices(services map[string]octaspace.ServiceOffer) string {
	var offered []string
	for name, s := range services {
		if s.Offered {
			offered = append(offered, name)
		}
	}
	if len(offered) == 0 {
		return "-"
	}
	sort.Strings(offered)
	return strings.Join(offered, ", ")
}

func serviceDetail(services map[string]octaspace.ServiceOffer) string {
	if len(services) == 0 {
		return "-"
	}
	names := make([]string, 0, len(services))
	for name := range services {
		names = append(names, name)
	}
	sort.Strings(names)
	parts := make([]string, 0, len(names))
	for _, name := range names {
		s := services[name]
		switch {
		case s.Offered:
			parts = append(parts, name+": offered")
		case s.Reason != "":
			parts = append(parts, name+": not offered ("+s.Reason+")")
		default:
			parts = append(parts, name+": not offered")
		}
	}
	return strings.Join(parts, "\n")
}

func usageSummary(u *octaspace.Usage) string {
	if u == nil {
		return "-"
	}
	return fmt.Sprintf("%s/%s", FormatBytes(u.Free), FormatBytes(u.Size))
}

func usageDetail(u *octaspace.Usage) string {
	if u == nil {
		return "-"
	}
	return fmt.Sprintf("%s free of %s (%.0f%% used)", FormatBytes(u.Free), FormatBytes(u.Size), u.UsedPercent)
}

func uptimePercent(p *float64) string {
	if p == nil {
		return "-"
	}
	return fmt.Sprintf("%.1f%%", *p)
}

// hostingGPUSummary summarizes the GPUs a node reports as open objects.
func hostingGPUSummary(gpus []map[string]any) string {
	if len(gpus) == 0 {
		return "-"
	}
	model, _ := gpus[0]["model"].(string)
	mem, _ := numberField(gpus[0], "mem_total_mb")
	return gpuSummary(len(gpus), model, int(mem))
}

func hostingGPUDetail(g map[string]any) string {
	model, _ := g["model"].(string)
	mem, _ := numberField(g, "mem_total_mb")
	out := gpuSummary(1, model, int(mem))
	var extra []string
	if v, ok := numberField(g, "gpu_temperature"); ok {
		extra = append(extra, fmt.Sprintf("%.0f°C", v))
	}
	if v, ok := numberField(g, "gpu_utilization"); ok {
		extra = append(extra, fmt.Sprintf("%.0f%% util", v))
	}
	if v, ok := numberField(g, "power_draw_watt"); ok {
		extra = append(extra, fmt.Sprintf("%.0f W", v))
	}
	if len(extra) > 0 {
		out += " (" + strings.Join(extra, ", ") + ")"
	}
	return out
}

// numberField reads a numeric field of an open GPU object; ok is false when
// it is absent, null or not a number.
func numberField(m map[string]any, key string) (float64, bool) {
	switch v := m[key].(type) {
	case float64:
		return v, !math.IsNaN(v)
	case string:
		f, err := strconv.ParseFloat(strings.TrimSpace(v), 64)
		return f, err == nil
	default:
		return 0, false
	}
}

func yesNo(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}

func orDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

var coresSuffixRe = regexp.MustCompile(` \d+-Cores?$`)

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
