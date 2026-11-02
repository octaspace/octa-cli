package cli

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	octaspace "github.com/octaspace/go-sdk"
	"github.com/octaspace/octa/internal/client"
	"github.com/octaspace/octa/internal/config"
	"github.com/octaspace/octa/internal/ui"
	"github.com/spf13/cobra"
)

var nodesCmd = &cobra.Command{
	Use:     "nodes",
	Aliases: []string{"hosting"},
	Short:   "Manage the nodes you host",
}

var nodesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List your online nodes",
	Long:  "List the account's online nodes. A node that is down is not listed.",
	Args:  noArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		c := client.New(cfg)
		format, _ := cmd.Flags().GetString("output")
		if format == "json" {
			out, err := c.Hosting.NodesRaw(cmd.Context())
			if err != nil {
				return client.Friendly(err)
			}
			fmt.Println(string(out))
			return nil
		}
		nodes, err := c.Hosting.Nodes(cmd.Context())
		if err != nil {
			return client.Friendly(err)
		}
		if len(nodes) == 0 {
			fmt.Println("No online nodes.")
			return nil
		}
		return ui.RenderHostingNodesTable(nodes)
	},
}

var nodesShowCmd = &cobra.Command{
	Use:   "show <number>",
	Short: "Show detailed information about one of your nodes",
	Args:  exactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		nodes, err := client.New(cfg).Hosting.Nodes(cmd.Context())
		if err != nil {
			return client.Friendly(err)
		}
		node, err := findHostingNode(nodes, args[0])
		if err != nil {
			return err
		}

		format, _ := cmd.Flags().GetString("output")
		if format == "json" {
			return printJSON(node)
		}
		return ui.RenderHostingNodeDetail(node)
	},
}

var nodesPricesCmd = &cobra.Command{
	Use:   "prices <number>",
	Short: "Update the prices of one of your nodes (in USD)",
	Long: `Update the prices of one of your nodes. Prices are in US dollars with up to
six decimal places; only the given ones change.

  --base      per hour of rent, $0.0001 .. $1000
  --storage   per GB of disk per hour, $0 .. $1000
  --traffic   per GB of traffic, $0.0001 .. $1000
  --min-rent  minimum rent of a compute session, 0..60 minutes (0 = off):
              a session stopped sooner is charged base and storage for it`,
	Example: `  octa nodes prices 7KQ4-M2XD-4 --base 0.25 --min-rent 15`,
	Args:    exactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		prices, err := nodePricesFromFlags(cmd)
		if err != nil {
			return err
		}

		cfg, err := config.Load()
		if err != nil {
			return err
		}

		number := strings.TrimSpace(args[0])
		if err := client.New(cfg).Hosting.SetPrices(cmd.Context(), number, prices); err != nil {
			return client.Friendly(err)
		}

		format, _ := cmd.Flags().GetString("output")
		if format == "json" {
			return printJSON(map[string]any{"status": "ok", "number": number, "prices": prices})
		}
		fmt.Printf("Prices updated for node %s.\n", number)
		return nil
	},
}

// nodePricesFromFlags reads the price flags that were given.
func nodePricesFromFlags(cmd *cobra.Command) (octaspace.NodePrices, error) {
	var prices octaspace.NodePrices
	var errs []error
	for _, f := range []struct {
		name string
		dst  **octaspace.MicroUSD
	}{
		{"base", &prices.Base},
		{"storage", &prices.Storage},
		{"traffic", &prices.Traffic},
	} {
		if !cmd.Flags().Changed(f.name) {
			continue
		}
		v, _ := cmd.Flags().GetString(f.name)
		m, err := parseUSD(v)
		if err != nil {
			errs = append(errs, fmt.Errorf("--%s: %w", f.name, err))
			continue
		}
		*f.dst = octaspace.Price(m)
	}
	if cmd.Flags().Changed("min-rent") {
		v, _ := cmd.Flags().GetInt("min-rent")
		if v < 0 || v > 60 {
			errs = append(errs, errors.New("--min-rent must be between 0 and 60 minutes"))
		}
		prices.MinRentMinutes = octaspace.Int(v)
	}
	if err := errors.Join(errs...); err != nil {
		return prices, err
	}
	if prices == (octaspace.NodePrices{}) {
		return prices, errors.New("at least one of --base, --storage, --traffic or --min-rent is required")
	}
	return prices, nil
}

// parseUSD parses a dollar amount such as "0.25" or "$1" into micro-USD
// exactly, without going through a float.
func parseUSD(s string) (octaspace.MicroUSD, error) {
	s = strings.TrimPrefix(strings.TrimSpace(s), "$")
	whole, frac, _ := strings.Cut(s, ".")
	if whole == "" && frac == "" {
		return 0, fmt.Errorf("%q is not a dollar amount", s)
	}
	if len(frac) > 6 {
		return 0, fmt.Errorf("%q has more than 6 decimal places", s)
	}
	if whole == "" {
		whole = "0"
	}
	digits := whole + frac + strings.Repeat("0", 6-len(frac))
	for _, r := range digits {
		if r < '0' || r > '9' {
			return 0, fmt.Errorf("%q is not a dollar amount", s)
		}
	}
	n, err := strconv.ParseInt(digits, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%q is out of range", s)
	}
	return octaspace.MicroUSD(n), nil
}

// findHostingNode finds a node by number, ignoring case, dashes and spaces
// and reading O as 0 and I, L as 1, as the API does.
func findHostingNode(nodes []octaspace.HostingNode, number string) (octaspace.HostingNode, error) {
	want := normalizeNodeNumber(number)
	for _, n := range nodes {
		if normalizeNodeNumber(n.Number) == want {
			return n, nil
		}
	}
	return octaspace.HostingNode{}, fmt.Errorf("no online node %q among your nodes", number)
}

func normalizeNodeNumber(s string) string {
	return strings.Map(func(r rune) rune {
		switch r {
		case '-', ' ':
			return -1
		case 'o', 'O':
			return '0'
		case 'i', 'I', 'l', 'L':
			return '1'
		}
		if r >= 'a' && r <= 'z' {
			return r - 'a' + 'A'
		}
		return r
	}, s)
}

func init() {
	nodesListCmd.Flags().StringP("output", "o", "table", "Output format: table or json")
	nodesShowCmd.Flags().StringP("output", "o", "table", "Output format: table or json")

	nodesPricesCmd.Flags().String("base", "", "Base price per hour, USD")
	nodesPricesCmd.Flags().String("storage", "", "Storage price per GB of disk per hour, USD")
	nodesPricesCmd.Flags().String("traffic", "", "Traffic price per GB, USD")
	nodesPricesCmd.Flags().Int("min-rent", 0, "Minimum rent of a compute session, minutes (0..60, 0 = off)")
	nodesPricesCmd.Flags().StringP("output", "o", "table", "Output format: table or json")

	nodesCmd.AddCommand(nodesListCmd)
	nodesCmd.AddCommand(nodesShowCmd)
	nodesCmd.AddCommand(nodesPricesCmd)
}
