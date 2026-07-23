package ui

import (
	"encoding/json"
	"math/big"
	"testing"

	octaspace "github.com/octaspace/go-sdk"
)

// TestSessionChargeAmountOverflow reproduces B4: a real session returned by the
// API carries charge_amount=31428028938060847852, which exceeds uint64's max
// (18446744073709551615). The previous uint64 field crashed json.Unmarshal for
// the whole list; octaspace.Session uses BigInt and must decode it cleanly.
func TestSessionChargeAmountOverflow(t *testing.T) {
	const payload = `[{"uuid":"550e8400-e29b-41d4-a716-446655440000","app_name":"jupyter","node_id":42,"duration":30,"charge_amount":31428028938060847852}]`

	var sessions []octaspace.Session
	if err := json.Unmarshal([]byte(payload), &sessions); err != nil {
		t.Fatalf("unmarshal failed on oversized charge_amount: %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("expected 1 session, got %d", len(sessions))
	}

	want, _ := new(big.Int).SetString("31428028938060847852", 10)
	if sessions[0].ChargeAmount.Int == nil || sessions[0].ChargeAmount.Cmp(want) != 0 {
		t.Fatalf("charge_amount = %v, want %s", sessions[0].ChargeAmount.Int, want)
	}

	// Rendering must not panic on the real value.
	if err := RenderSessionsTable(sessions); err != nil {
		t.Fatalf("RenderSessionsTable: %v", err)
	}
}

func TestFormatOCTA(t *testing.T) {
	cases := []struct {
		wei  string
		prec int
		want string
	}{
		{"31428028938060847852", 6, "31.428029 OCTA"},
		{"1000000000000000000", 4, "1.0000 OCTA"},
		{"0", 4, "0.0000 OCTA"},
	}
	for _, tc := range cases {
		wei, _ := new(big.Int).SetString(tc.wei, 10)
		if got := FormatOCTA(wei, tc.prec); got != tc.want {
			t.Errorf("FormatOCTA(%s, %d) = %q, want %q", tc.wei, tc.prec, got, tc.want)
		}
	}

	// A nil value (absent charge_amount) must be treated as zero, not panic.
	if got := FormatOCTA(nil, 2); got != "0.00 OCTA" {
		t.Errorf("FormatOCTA(nil, 2) = %q, want %q", got, "0.00 OCTA")
	}
}

func TestFormatUSD(t *testing.T) {
	cases := []struct {
		wei         string
		marketPrice float64
		want        string
	}{
		{"742404000000000000", 0.05, "0.0371"},
		{"1000000000000000000", 0.1, "0.1000"},
		{"0", 0.1, "0.0000"},
	}
	for _, tc := range cases {
		wei, _ := new(big.Int).SetString(tc.wei, 10)
		if got := FormatUSD(wei, tc.marketPrice); got != tc.want {
			t.Errorf("FormatUSD(%s, %v) = %q, want %q", tc.wei, tc.marketPrice, got, tc.want)
		}
	}

	// A nil value (absent charge_amount) must be treated as zero, not panic.
	if got := FormatUSD(nil, 0.1); got != "0.0000" {
		t.Errorf("FormatUSD(nil, 0.1) = %q, want %q", got, "0.0000")
	}
}

func TestFormatMbps(t *testing.T) {
	cases := []struct {
		mbps float64
		want string
	}{
		{0, "0 bps"},
		{0.5, "500 Kbps"},
		{2892312.0 / 125000, "23.1 Mbps"},   // real relay net_down_mbits normalized
		{62118848.0 / 125000, "497.0 Mbps"}, // real relay net_up_mbits normalized
		{1500, "1.5 Gbps"},
	}
	for _, tc := range cases {
		if got := formatMbps(tc.mbps); got != tc.want {
			t.Errorf("formatMbps(%v) = %q, want %q", tc.mbps, got, tc.want)
		}
	}
}
