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
		if err == nil || !strings.Contains(err.Error(), "no running session found") {
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

func apiErr(status int, msg string) *octaspace.APIError {
	return &octaspace.APIError{StatusCode: status, Message: msg}
}

func TestFriendlyMessages(t *testing.T) {
	cases := []struct {
		err  error
		want string
	}{
		{&octaspace.AuthenticationError{APIError: apiErr(401, "Missing bearer token")}, "needs an API key"},
		{&octaspace.AuthenticationError{APIError: apiErr(401, "Invalid bearer token")}, "octa_ak_"},
		{&octaspace.NotFoundError{APIError: apiErr(404, "Session not found")}, "Session not found"},
		{&octaspace.NotFoundError{APIError: apiErr(404, "")}, "resource not found"},
		{&octaspace.ValidationError{APIError: apiErr(400, "Insufficient balance")}, "Insufficient balance"},
		{&octaspace.ConflictError{APIError: apiErr(409, "Session is still starting or restarting")}, "still starting"},
		{&octaspace.RateLimitError{APIError: &octaspace.APIError{StatusCode: 429, Message: "Too many requests", RetryAfter: 30}}, "Too many requests — retry after 30s"},
		{&octaspace.RateLimitError{APIError: &octaspace.APIError{StatusCode: 429, Message: "Session was restarted too recently, retry in 288 seconds", RetryAfter: 288}}, "recently, retry in 288 seconds"},
		{&octaspace.ServerError{APIError: apiErr(503, "Service is temporarily unavailable")}, "Service is temporarily unavailable — try again later"},
		{&octaspace.ServerError{APIError: apiErr(503, "")}, "service temporarily unavailable: try again later"},
	}
	for _, tc := range cases {
		got := Friendly(tc.err)
		if got != nil && strings.Count(got.Error(), "retry") > 1 {
			t.Errorf("Friendly(%v) repeats the wait: %v", tc.err, got)
		}
		if got == nil || !strings.Contains(got.Error(), tc.want) {
			t.Errorf("Friendly(%v) = %v, want it to contain %q", tc.err, got, tc.want)
		}
	}
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
