package cli

import (
	"strings"
	"testing"

	octaspace "github.com/octaspace/go-sdk"
)

func TestStopTargets(t *testing.T) {
	running := []octaspace.Session{{UUID: "aaaa-1"}, {UUID: "abbb-2"}, {UUID: "cccc-3"}}

	all, err := stopTargets(running, nil)
	if err != nil || len(all) != 3 {
		t.Fatalf("--all = %v, %v", all, err)
	}

	got, err := stopTargets(running, []string{"cc", "aaaa", "aaaa-1"})
	if err != nil || len(got) != 2 || got[0].UUID != "cccc-3" || got[1].UUID != "aaaa-1" {
		t.Errorf("targets = %v, %v", got, err)
	}

	if _, err := stopTargets(running, []string{"cccc", "a"}); err == nil || !strings.Contains(err.Error(), "ambiguous") {
		t.Errorf("ambiguous prefix: err = %v", err)
	}
	if _, err := stopTargets(running, []string{"cccc", "zz"}); err == nil {
		t.Error("an unknown UUID must fail the whole call")
	}
}

func TestSessionsStopArgs(t *testing.T) {
	var api fakeAPI
	api.start(t)
	if err := runCLI(t, "sessions", "stop"); err == nil || !strings.Contains(err.Error(), "--all") {
		t.Errorf("no args: err = %v", err)
	}
	if err := runCLI(t, "sessions", "stop", "--all", "aaaa"); err == nil || !strings.Contains(err.Error(), "takes no UUIDs") {
		t.Errorf("--all with UUIDs: err = %v", err)
	}
	if api.body != nil {
		t.Errorf("request sent: %s", api.body)
	}
}

func TestMaskKey(t *testing.T) {
	cases := map[string]string{
		"":                             "",
		"octa_ak_GUVzn8NyEumGoccfcKXu": "octa_ak_GUVz…",
		"octa_ak_ab":                   "**********",
		"legacy-token-123":             "****************",
	}
	for in, want := range cases {
		if got := maskKey(in); got != want {
			t.Errorf("maskKey(%q) = %q, want %q", in, got, want)
		}
	}
}
