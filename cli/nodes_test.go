package cli

import (
	"strings"
	"testing"

	octaspace "github.com/octaspace/go-sdk"
	"github.com/spf13/cobra"
)

func TestParseUSD(t *testing.T) {
	ok := map[string]octaspace.MicroUSD{
		"0.25": 250000, "$1": 1000000, ".5": 500000, "0.0001": 100,
		"1000": 1000000000, "0.000001": 1, " 2.5 ": 2500000, "3.": 3000000,
	}
	for in, want := range ok {
		got, err := parseUSD(in)
		if err != nil || got != want {
			t.Errorf("parseUSD(%q) = %d, %v; want %d", in, got, err, want)
		}
	}
	for _, in := range []string{"", ".", "abc", "1.0000001", "-1", "1e3", "1,5"} {
		if _, err := parseUSD(in); err == nil {
			t.Errorf("parseUSD(%q) must fail", in)
		}
	}
}

func TestNormalizeNodeNumber(t *testing.T) {
	nodes := []octaspace.HostingNode{{Number: "M95S-WZG8-7"}, {Number: "7KQ4-M2XD-4"}}
	for _, q := range []string{"m95s wzg8 7", "M95SWZG87", "7kq4-m2xd-4"} {
		if _, err := findHostingNode(nodes, q); err != nil {
			t.Errorf("findHostingNode(%q): %v", q, err)
		}
	}
	if got := normalizeNodeNumber("oIl-0"); got != "0110" {
		t.Errorf("normalize = %q", got)
	}
	if _, err := findHostingNode(nodes, "0000-0000-0"); err == nil {
		t.Error("unknown node must fail")
	}
}

func pricesCmd(t *testing.T, args ...string) *cobra.Command {
	t.Helper()
	cmd := &cobra.Command{}
	cmd.Flags().String("base", "", "")
	cmd.Flags().String("storage", "", "")
	cmd.Flags().String("traffic", "", "")
	cmd.Flags().Int("min-rent", 0, "")
	if err := cmd.Flags().Parse(args); err != nil {
		t.Fatal(err)
	}
	return cmd
}

func TestNodePricesFromFlags(t *testing.T) {
	prices, err := nodePricesFromFlags(pricesCmd(t, "--base", "0.25", "--min-rent", "0"))
	if err != nil {
		t.Fatal(err)
	}
	if prices.Base == nil || *prices.Base != 250000 || prices.Storage != nil || prices.Traffic != nil {
		t.Errorf("prices = %+v", prices)
	}
	if prices.MinRentMinutes == nil || *prices.MinRentMinutes != 0 {
		t.Error("--min-rent 0 must be sent to turn the minimum off")
	}

	if _, err := nodePricesFromFlags(pricesCmd(t)); err == nil || !strings.Contains(err.Error(), "at least one") {
		t.Errorf("no flags: err = %v", err)
	}
	if _, err := nodePricesFromFlags(pricesCmd(t, "--min-rent", "61", "--traffic", "x")); err == nil ||
		!strings.Contains(err.Error(), "--min-rent") || !strings.Contains(err.Error(), "--traffic") {
		t.Errorf("invalid flags: err = %v", err)
	}
}

func TestMinRentWarning(t *testing.T) {
	if w := minRentWarning(octaspace.Session{MinRentMinutes: 15, Duration: 120}); !strings.Contains(w, "15-minute minimum") {
		t.Errorf("warning = %q", w)
	}
	if w := minRentWarning(octaspace.Session{MinRentMinutes: 15, Duration: 15 * 60}); w != "" {
		t.Errorf("past the minimum: %q", w)
	}
	if w := minRentWarning(octaspace.Session{Duration: 10}); w != "" {
		t.Errorf("no minimum: %q", w)
	}
}
