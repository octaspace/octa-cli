package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	octaspace "github.com/octaspace/go-sdk"
	"github.com/octaspace/octa/internal/client"
	"github.com/spf13/cobra"
)

// noArgs rejects positional arguments. On a command with subcommands an
// argument is most likely a mistyped subcommand; elsewhere it usually comes
// from an unquoted value with spaces (e.g. --node 7KQ4 M2XD 4).
func noArgs(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return nil
	}
	if cmd.HasSubCommands() {
		msg := fmt.Sprintf("unknown command %q for %q", args[0], cmd.CommandPath())
		if cmd.SuggestionsMinimumDistance <= 0 {
			cmd.SuggestionsMinimumDistance = 2 // cobra sets it on the root only
		}
		if s := cmd.SuggestionsFor(args[0]); len(s) > 0 {
			msg += "\nDid you mean " + strings.Join(s, " or ") + "?"
		}
		return fmt.Errorf("%s\nRun '%s --help' for usage", msg, cmd.CommandPath())
	}
	return fmt.Errorf("unexpected argument(s) %q: quote flag values that contain spaces\nUsage: %s", args, cmd.UseLine())
}

func nonEmpty(values []string) []string {
	var out []string
	for _, v := range values {
		if v = strings.TrimSpace(v); v != "" {
			out = append(out, v)
		}
	}
	return out
}

func orDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

// batchFileItem is one request of a --from-file batch: the API's
// ComputeRequest fields plus an optional id (string or number).
type batchFileItem struct {
	ID json.RawMessage `json:"id"`
	octaspace.ComputeStartParams
}

// loadComputeBatchFile reads a JSON array of compute start requests, in the
// API's field names. A missing id becomes the 1-based position.
func loadComputeBatchFile(path string) ([]octaspace.ComputeBatchItem, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("--from-file: %w", err)
	}
	return parseComputeBatch(data)
}

func parseComputeBatch(data []byte) ([]octaspace.ComputeBatchItem, error) {
	var raw []json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("--from-file: expected a JSON array of start requests: %w", err)
	}
	if len(raw) == 0 {
		return nil, errors.New("--from-file: the batch is empty")
	}

	items := make([]octaspace.ComputeBatchItem, 0, len(raw))
	seen := map[string]bool{}
	var errs []error
	for i, r := range raw {
		var it batchFileItem
		dec := json.NewDecoder(bytes.NewReader(r))
		dec.DisallowUnknownFields() // the API rejects unknown fields too
		if err := dec.Decode(&it); err != nil {
			errs = append(errs, fmt.Errorf("request %d: %w", i+1, err))
			continue
		}
		id, err := batchID(it.ID, i)
		if err != nil {
			errs = append(errs, fmt.Errorf("request %d: %w", i+1, err))
			continue
		}
		if seen[id] {
			errs = append(errs, fmt.Errorf("request %d: duplicate id %q", i+1, id))
		}
		seen[id] = true
		for _, e := range checkStartParams(&it.ComputeStartParams) {
			errs = append(errs, fmt.Errorf("request %s: %w", id, e))
		}
		items = append(items, octaspace.ComputeBatchItem{ID: id, ComputeStartParams: it.ComputeStartParams})
	}
	if err := errors.Join(errs...); err != nil {
		return nil, err
	}
	return items, nil
}

// batchID returns the request id as a string: a JSON string or number, or the
// 1-based position when absent.
func batchID(raw json.RawMessage, index int) (string, error) {
	raw = bytes.TrimSpace(raw)
	switch {
	case len(raw) == 0 || string(raw) == "null":
		return strconv.Itoa(index + 1), nil
	case raw[0] == '"':
		var s string
		if err := json.Unmarshal(raw, &s); err != nil || s == "" {
			return "", errors.New("id must be a non-empty string or a number")
		}
		return s, nil
	default:
		var n json.Number
		if err := json.Unmarshal(raw, &n); err != nil {
			return "", errors.New("id must be a string or a number")
		}
		return n.String(), nil
	}
}

// batchOutcome is one line of a batch report.
type batchOutcome struct {
	ID      string             `json:"id"`
	Node    string             `json:"node_number,omitempty"`
	Status  int                `json:"status"` // 0 - started, 1 - failed
	UUID    string             `json:"uuid,omitempty"`
	Reason  string             `json:"reason,omitempty"`
	Session *octaspace.Session `json:"session,omitempty"` // with --wait
}

// reportBatch prints the results of a batch start and, with --wait, waits for
// every started session. It returns an error when any request failed.
func reportBatch(cmd *cobra.Command, c *octaspace.Client, results []octaspace.BatchResult, nodes map[string]string) error {
	format, _ := cmd.Flags().GetString("output")
	wait, _ := cmd.Flags().GetBool("wait")
	timeout, _ := cmd.Flags().GetDuration("wait-timeout")
	jsonOut := format == "json"

	outcomes := make([]batchOutcome, 0, len(results))
	failed := 0
	for _, r := range results {
		o := batchOutcome{ID: r.ID, Node: nodes[r.ID], Status: r.Status, UUID: r.UUID, Reason: r.Reason}
		if !r.OK() {
			failed++
			if o.Status == 0 {
				o.Status = 1
			}
		}
		outcomes = append(outcomes, o)
	}

	if !jsonOut {
		printBatchOutcomes(outcomes)
	}

	if wait {
		for i := range outcomes {
			o := &outcomes[i]
			if o.UUID == "" {
				continue
			}
			if !jsonOut {
				fmt.Fprintf(os.Stderr, "Waiting for %s (%s):\n", o.UUID, orDash(o.Node))
			}
			s, err := waitReady(cmd.Context(), c, o.UUID, timeout, !jsonOut)
			if err != nil {
				failed++
				o.Status, o.Reason = 1, err.Error()
				if !jsonOut {
					fmt.Fprintf(os.Stderr, "  %v\n", err)
				}
				continue
			}
			o.Session = s
			if !jsonOut {
				fmt.Printf("\n%s (%s): ", s.UUID, orDash(s.NodeNumber))
				printSessionAccess(*s)
			}
		}
	}

	if jsonOut {
		if err := printJSON(outcomes); err != nil {
			return err
		}
	}
	if failed > 0 {
		return fmt.Errorf("%d of %d requests failed", failed, len(outcomes))
	}
	return nil
}

func printBatchOutcomes(outcomes []batchOutcome) {
	for _, o := range outcomes {
		if o.UUID != "" {
			fmt.Printf("[%s] %-12s started  %s\n", o.ID, orDash(o.Node), o.UUID)
		} else {
			fmt.Printf("[%s] %-12s failed   %s\n", o.ID, orDash(o.Node), orDash(o.Reason))
		}
	}
}

// startRenderBatch rents several render nodes in one request.
func startRenderBatch(cmd *cobra.Command, c *octaspace.Client, nodes []string) error {
	items := make([]octaspace.RenderBatchItem, len(nodes))
	byID := make(map[string]string, len(nodes))
	for i, n := range nodes {
		id := strconv.Itoa(i + 1)
		items[i] = octaspace.RenderBatchItem{ID: id, NodeNumber: n}
		byID[id] = n
	}
	results, err := c.Render.StartBatch(cmd.Context(), items)
	if err != nil {
		return client.Friendly(err)
	}
	return reportBatch(cmd, c, results, byID)
}
