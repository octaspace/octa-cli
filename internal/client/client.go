// Package client builds and configures the OctaSpace SDK client for the CLI
// and provides small helpers shared across commands.
package client

import (
	"errors"
	"fmt"
	"strings"

	octaspace "github.com/octaspace/go-sdk"
	"github.com/octaspace/octa/internal/config"
)

// userAgent identifies the CLI in outgoing requests.
const userAgent = "octa-cli"

// New builds an OctaSpace SDK client from the loaded config. The SDK sends the
// key as "Authorization: Bearer <key>"; an empty key makes anonymous requests.
func New(cfg *config.Config) *octaspace.Client {
	opts := []octaspace.Option{octaspace.WithUserAgent(userAgent)}
	if u := cfg.BaseURL(); u != "" {
		opts = append(opts, octaspace.WithBaseURL(u))
	}
	return octaspace.NewClient(cfg.APIKey, opts...)
}

// Friendly maps SDK error types to concise, user-facing messages, keeping the
// API's own message where it explains the problem. Unknown errors are
// returned unchanged.
func Friendly(err error) error {
	if err == nil {
		return nil
	}
	msg := apiMessage(err)
	switch {
	case octaspace.IsAuthError(err):
		if strings.Contains(strings.ToLower(msg), "missing") {
			return errors.New("this command needs an API key — run 'octa auth <key>'")
		}
		return errors.New("invalid API key — create an octa_ak_… key in OctaSpace (Profile → Credentials) and run 'octa auth <key>'")
	case octaspace.IsRateLimit(err):
		var rl *octaspace.RateLimitError
		if strings.Contains(strings.ToLower(msg), "retry in") {
			return errors.New(msg) // the API already says how long to wait
		}
		if errors.As(err, &rl) && rl.RetryAfter > 0 {
			return fmt.Errorf("%s — retry after %ds", orDefault(msg, "rate limited"), rl.RetryAfter)
		}
		return fmt.Errorf("%s — try again shortly", orDefault(msg, "rate limited"))
	case octaspace.IsNotFound(err):
		return errors.New(orDefault(msg, "resource not found"))
	case octaspace.IsConflict(err), octaspace.IsValidation(err):
		if msg != "" {
			return errors.New(msg)
		}
		return err
	}
	var se *octaspace.ServerError
	if errors.As(err, &se) && se.StatusCode == 503 {
		if strings.Contains(strings.ToLower(msg), "unavailable") {
			return fmt.Errorf("%s — try again later", msg)
		}
		return fmt.Errorf("service temporarily unavailable: %s", orDefault(msg, "try again later"))
	}
	return err
}

// apiMessage returns the message of the API error in err's chain, if any.
func apiMessage(err error) string {
	var apiErr *octaspace.APIError
	if errors.As(err, &apiErr) {
		return strings.TrimSpace(apiErr.Message)
	}
	return ""
}

func orDefault(s, def string) string {
	if s == "" {
		return def
	}
	return s
}

// MatchSession resolves a full or partial UUID against a session list. It
// returns an error when nothing matches or when the prefix is ambiguous.
// The API lists running sessions only, so a stopped session never matches.
func MatchSession(sessions []octaspace.Session, query string) (octaspace.Session, error) {
	var matched []octaspace.Session
	for _, s := range sessions {
		if s.UUID == query || strings.HasPrefix(s.UUID, query) {
			matched = append(matched, s)
		}
	}

	switch len(matched) {
	case 1:
		return matched[0], nil
	case 0:
		return octaspace.Session{}, fmt.Errorf("no running session found matching %q", query)
	default:
		var b strings.Builder
		fmt.Fprintf(&b, "ambiguous UUID %q matches %d sessions:", query, len(matched))
		for _, s := range matched {
			fmt.Fprintf(&b, "\n  %s", s.UUID)
		}
		return octaspace.Session{}, errors.New(b.String())
	}
}
