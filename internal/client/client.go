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

// New builds an OctaSpace SDK client from the loaded config.
func New(cfg *config.Config) *octaspace.Client {
	return octaspace.NewClient(cfg.APIKey, octaspace.WithUserAgent(userAgent))
}

// Friendly maps SDK error types to concise, user-facing messages. Unknown
// errors are returned unchanged.
func Friendly(err error) error {
	switch {
	case err == nil:
		return nil
	case octaspace.IsAuthError(err):
		return errors.New("invalid or expired token — run 'octa auth <token>'")
	case octaspace.IsNotFound(err):
		return errors.New("resource not found")
	case octaspace.IsRateLimit(err):
		var rl *octaspace.RateLimitError
		if errors.As(err, &rl) && rl.RetryAfter > 0 {
			return fmt.Errorf("rate limited — retry after %ds", rl.RetryAfter)
		}
		return errors.New("rate limited — try again shortly")
	default:
		return err
	}
}

// MatchSession resolves a full or partial UUID against a session list. It
// returns an error when nothing matches or when the prefix is ambiguous.
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
		return octaspace.Session{}, fmt.Errorf("no session found matching %q", query)
	default:
		var b strings.Builder
		fmt.Fprintf(&b, "ambiguous UUID %q matches %d sessions:", query, len(matched))
		for _, s := range matched {
			fmt.Fprintf(&b, "\n  %s", s.UUID)
		}
		return octaspace.Session{}, errors.New(b.String())
	}
}
