package client

import (
	"errors"
	"strings"
	"testing"

	octaspace "github.com/octaspace/go-sdk"
)

func sessions(uuids ...string) []octaspace.Session {
	out := make([]octaspace.Session, len(uuids))
	for i, u := range uuids {
		out[i] = octaspace.Session{UUID: u}
	}
	return out
}

func TestMatchSession(t *testing.T) {
	list := sessions(
		"550e8400-e29b-41d4-a716-446655440000",
		"550e8400-ffff-41d4-a716-446655440000",
		"1234abcd-e29b-41d4-a716-446655440000",
	)

	t.Run("exact full uuid", func(t *testing.T) {
		got, err := MatchSession(list, "1234abcd-e29b-41d4-a716-446655440000")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.UUID != "1234abcd-e29b-41d4-a716-446655440000" {
			t.Fatalf("got %q", got.UUID)
		}
	})

	t.Run("unique prefix", func(t *testing.T) {
		got, err := MatchSession(list, "1234")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.UUID != "1234abcd-e29b-41d4-a716-446655440000" {
			t.Fatalf("got %q", got.UUID)
		}
	})

	t.Run("no match", func(t *testing.T) {
		_, err := MatchSession(list, "deadbeef")
		if err == nil || !strings.Contains(err.Error(), "no session found") {
			t.Fatalf("want no-match error, got %v", err)
		}
	})

	t.Run("ambiguous prefix", func(t *testing.T) {
		_, err := MatchSession(list, "550e8400")
		if err == nil || !strings.Contains(err.Error(), "ambiguous") {
			t.Fatalf("want ambiguous error, got %v", err)
		}
	})
}

func TestFriendly(t *testing.T) {
	if Friendly(nil) != nil {
		t.Fatal("Friendly(nil) must be nil")
	}
	// Unknown errors pass through unchanged.
	sentinel := errors.New("some transport failure")
	if got := Friendly(sentinel); got != sentinel {
		t.Fatalf("Friendly should pass through unknown errors, got %v", got)
	}
}
