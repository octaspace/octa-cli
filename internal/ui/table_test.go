package ui

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"testing"

	octaspace "github.com/octaspace/go-sdk"
)

// loadFixture decodes testdata/<name>.json, a live response of the next API
// (IPs redacted), shared with the go-sdk test fixtures.
func loadFixture(t *testing.T, name string, dst any) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name+".json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, dst); err != nil {
		t.Fatalf("decode %s: %v", name, err)
	}
}

func TestFormatUSD(t *testing.T) {
	cases := []struct {
		in   octaspace.MicroUSD
		want string
	}{
		{0, "$0.00"},
		{100, "$0.0001"},
		{100000, "$0.10"},
		{2175268, "$2.175268"},
		{12000000, "$12.00"},
		{-1500000, "-$1.50"},
		{math.MinInt64, "-$9223372036854.775808"},
	}
	for _, tc := range cases {
		if got := FormatUSD(tc.in); got != tc.want {
			t.Errorf("FormatUSD(%d) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestFormatBalance(t *testing.T) {
	cases := []struct {
		amount   int64
		decimals int
		currency string
		want     string
	}{
		{2175268, 6, "USD", "$2.175268"},
		{-250, 2, "USD", "-$2.50"},
		{5, 0, "USD", "$5"},
		{1234, 3, "EUR", "1.234 EUR"},
	}
	for _, tc := range cases {
		if got := FormatBalance(tc.amount, tc.decimals, tc.currency); got != tc.want {
			t.Errorf("FormatBalance(%d, %d, %q) = %q, want %q", tc.amount, tc.decimals, tc.currency, got, tc.want)
		}
	}
}

func TestFormatNet(t *testing.T) {
	cases := []struct {
		down, up int
		want     string
	}{
		{0, 0, "-"},
		{240, 466, "240 Mbps/466 Mbps"},
		{1500, 0, "1.5 Gbps/-"},
	}
	for _, tc := range cases {
		if got := FormatNet(tc.down, tc.up); got != tc.want {
			t.Errorf("FormatNet(%d, %d) = %q, want %q", tc.down, tc.up, got, tc.want)
		}
	}
}

func TestFormatDuration(t *testing.T) {
	if got := FormatDuration(3*3600 + 7*60 + 5); got != "3h 07m" {
		t.Errorf("FormatDuration = %q", got)
	}
	if got := FormatDuration(2*86400 + 3*3600); got != "2d 03h" {
		t.Errorf("FormatDuration = %q", got)
	}
}

func TestSessionStatus(t *testing.T) {
	var ready, starting octaspace.Session
	loadFixture(t, "session_compute_ready", &ready)
	loadFixture(t, "session_compute_starting", &starting)

	if got := SessionStatus(ready); got != "ready" {
		t.Errorf("ready status = %q", got)
	}
	pct := 42
	starting.Progress, starting.PullPercent = "Pulling image", &pct
	if got := SessionStatus(starting); got != "Pulling image (42%)" {
		t.Errorf("starting status = %q", got)
	}
	if got := SessionStatus(octaspace.Session{Status: "starting"}); got != "starting" {
		t.Errorf("bare starting status = %q", got)
	}
}

func TestSSHAddress(t *testing.T) {
	if got := SSHAddress(nil); got != "" {
		t.Errorf("nil endpoint = %q", got)
	}
	e := &octaspace.SSHEndpoint{Host: "203.0.113.10", Port: 31022}
	if got := SSHAddress(e); got != "root@203.0.113.10 -p 31022" {
		t.Errorf("default user = %q", got)
	}
	e.User = "ubuntu"
	if got := SSHAddress(e); got != "ubuntu@203.0.113.10 -p 31022" {
		t.Errorf("image user = %q", got)
	}
}

func TestSessionRow(t *testing.T) {
	var s octaspace.Session
	loadFixture(t, "session_compute_ready", &s)
	row := sessionToRow(s)
	want := []string{"2a63b54f", "compute", "alpine:3.20", "M95S-WZG8-7", "ready", "0h 00m", "$0.00", "-", "-"}
	for i := range want {
		if row[i] != want[i] {
			t.Errorf("column %d = %q, want %q", i, row[i], want[i])
		}
	}
}

func TestComputeMachineRow(t *testing.T) {
	var machines []octaspace.ComputeMachine
	loadFixture(t, "services_compute", &machines)
	if len(machines) == 0 {
		t.Fatal("empty fixture")
	}
	for _, m := range machines {
		row := computeMachineRow(m)
		if row[0] != m.NodeNumber || row[6] != FormatUSD(m.BasePrice) {
			t.Errorf("row = %q", row)
		}
	}
	if err := RenderComputeTable(machines); err != nil {
		t.Fatal(err)
	}
}

func TestRenderTables(t *testing.T) {
	var render []octaspace.RenderMachine
	loadFixture(t, "services_render", &render)
	if err := RenderRenderTable(render); err != nil {
		t.Fatal(err)
	}

	var nodes []octaspace.HostingNode
	loadFixture(t, "hosting_nodes", &nodes)
	if len(nodes) == 0 {
		t.Fatal("empty hosting fixture")
	}
	if err := RenderHostingNodesTable(nodes); err != nil {
		t.Fatal(err)
	}
	for _, n := range nodes {
		if err := RenderHostingNodeDetail(n); err != nil {
			t.Fatal(err)
		}
	}
}

func TestHostingGPUSummary(t *testing.T) {
	gpus := []map[string]any{
		{"model": "NVIDIA GeForce RTX 3090", "mem_total_mb": float64(24576), "gpu_temperature": float64(47)},
		{"model": "NVIDIA GeForce RTX 3090", "mem_total_mb": float64(24576)},
	}
	if got := hostingGPUSummary(gpus); got != "2x RTX 3090 24GB" {
		t.Errorf("summary = %q", got)
	}
	// Values a node does not know may arrive as nil or as the string "null".
	g := map[string]any{"model": "Intel Arc B580", "mem_total_mb": "12288", "gpu_temperature": nil, "power_draw_watt": "null"}
	if got := hostingGPUDetail(g); got != "Arc B580 12GB" {
		t.Errorf("detail = %q", got)
	}
}

func TestOfferedServices(t *testing.T) {
	services := map[string]octaspace.ServiceOffer{
		"render":  {Offered: true},
		"compute": {Offered: true},
		"vpn":     {Offered: false, Reason: "ports_closed"},
	}
	if got := OfferedServices(services); got != "compute, render" {
		t.Errorf("offered = %q", got)
	}
	if got := OfferedServices(nil); got != "-" {
		t.Errorf("none offered = %q", got)
	}
}
