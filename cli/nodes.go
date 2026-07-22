package cli

import (
	"errors"
	"fmt"
	"os"
	"strconv"

	octaspace "github.com/octaspace/go-sdk"
	"github.com/octaspace/octa/internal/client"
	"github.com/octaspace/octa/internal/config"
	"github.com/octaspace/octa/internal/ui"
	"github.com/spf13/cobra"
)

var nodesCmd = &cobra.Command{
	Use:   "nodes",
	Short: "Manage nodes",
}

var nodesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all nodes",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		params := nodesListParams(cmd)

		c := client.New(cfg)
		format, _ := cmd.Flags().GetString("output")
		if format == "json" {
			out, err := c.Nodes.ListRaw(cmd.Context(), params)
			if err != nil {
				return client.Friendly(err)
			}
			fmt.Println(string(out))
			return nil
		}
		nodes, err := c.Nodes.List(cmd.Context(), params)
		if err != nil {
			return client.Friendly(err)
		}

		return ui.RenderNodesTable(nodes)
	},
}

var nodesShowCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Show detailed information about a node",
	Args:  exactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := parseNodeID(args[0])
		if err != nil {
			return err
		}

		cfg, err := config.Load()
		if err != nil {
			return err
		}

		c := client.New(cfg)
		node, err := c.Nodes.Find(cmd.Context(), id)
		if err != nil {
			return client.Friendly(err)
		}

		format, _ := cmd.Flags().GetString("output")
		if format == "json" {
			return printJSON(node)
		}

		return ui.RenderNodeDetail(*node)
	},
}

var nodesIdentCmd = &cobra.Command{
	Use:   "ident <id>",
	Short: "Download a node's identity file",
	Args:  exactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := parseNodeID(args[0])
		if err != nil {
			return err
		}

		cfg, err := config.Load()
		if err != nil {
			return err
		}

		c := client.New(cfg)
		data, err := c.Nodes.DownloadIdent(cmd.Context(), id)
		if err != nil {
			return client.Friendly(err)
		}

		out, _ := cmd.Flags().GetString("out")
		if out == "" {
			out = fmt.Sprintf("node-%d-ident.bin", id)
		}
		return writeDownload(out, data)
	},
}

var nodesLogsCmd = &cobra.Command{
	Use:   "logs <id>",
	Short: "Download a node's log archive",
	Args:  exactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := parseNodeID(args[0])
		if err != nil {
			return err
		}

		cfg, err := config.Load()
		if err != nil {
			return err
		}

		c := client.New(cfg)
		data, err := c.Nodes.DownloadLogs(cmd.Context(), id)
		if err != nil {
			return client.Friendly(err)
		}

		out, _ := cmd.Flags().GetString("out")
		if out == "" {
			out = fmt.Sprintf("node-%d-logs.bin", id)
		}
		return writeDownload(out, data)
	},
}

var nodesPricesCmd = &cobra.Command{
	Use:   "prices <id>",
	Short: "Update a node's prices (values in USD cents)",
	Args:  exactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := parseNodeID(args[0])
		if err != nil {
			return err
		}

		prices := map[string]any{}
		if cmd.Flags().Changed("base") {
			v, _ := cmd.Flags().GetInt64("base")
			prices["base"] = v
		}
		if cmd.Flags().Changed("storage") {
			v, _ := cmd.Flags().GetInt64("storage")
			prices["storage"] = v
		}
		if cmd.Flags().Changed("traffic") {
			v, _ := cmd.Flags().GetInt64("traffic")
			prices["traffic"] = v
		}
		if len(prices) == 0 {
			return errors.New("at least one of --base, --storage or --traffic is required")
		}

		cfg, err := config.Load()
		if err != nil {
			return err
		}

		c := client.New(cfg)
		if err := c.Nodes.UpdatePrices(cmd.Context(), id, prices); err != nil {
			return client.Friendly(err)
		}

		format, _ := cmd.Flags().GetString("output")
		if format == "json" {
			return printJSON(map[string]any{"status": "ok", "node_id": id, "prices": prices})
		}
		fmt.Printf("Prices updated for node %d.\n", id)
		return nil
	},
}

var nodesRebootCmd = &cobra.Command{
	Use:   "reboot <id>",
	Short: "Reboot a node",
	Args:  exactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := parseNodeID(args[0])
		if err != nil {
			return err
		}

		cfg, err := config.Load()
		if err != nil {
			return err
		}

		c := client.New(cfg)
		if err := c.Nodes.Reboot(cmd.Context(), id); err != nil {
			return client.Friendly(err)
		}

		format, _ := cmd.Flags().GetString("output")
		if format == "json" {
			return printJSON(map[string]any{"status": "ok", "node_id": id})
		}
		fmt.Printf("Reboot triggered for node %d.\n", id)
		return nil
	},
}

func nodesListParams(cmd *cobra.Command) *octaspace.ListNodesParams {
	var params *octaspace.ListNodesParams
	if cmd.Flags().Changed("country") {
		v, _ := cmd.Flags().GetString("country")
		if params == nil {
			params = &octaspace.ListNodesParams{}
		}
		params.Country = &v
	}
	if cmd.Flags().Changed("app") {
		v, _ := cmd.Flags().GetString("app")
		if params == nil {
			params = &octaspace.ListNodesParams{}
		}
		params.AppUUID = &v
	}
	return params
}

func parseNodeID(s string) (int64, error) {
	id, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid node ID %q", s)
	}
	return id, nil
}

func writeDownload(path string, data []byte) error {
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return err
	}
	fmt.Printf("Wrote %d bytes to %s\n", len(data), path)
	return nil
}

func init() {
	nodesListCmd.Flags().StringP("output", "o", "table", "Output format: table or json")
	nodesListCmd.Flags().String("country", "", "Filter by ISO country code (e.g. US, DE)")
	nodesListCmd.Flags().String("app", "", "Filter by compatible application UUID")

	nodesIdentCmd.Flags().StringP("out", "o", "", "Output file path (default node-<id>-ident.bin)")
	nodesLogsCmd.Flags().StringP("out", "o", "", "Output file path (default node-<id>-logs.bin)")

	nodesShowCmd.Flags().StringP("output", "o", "table", "Output format: table or json")

	nodesPricesCmd.Flags().Int64("base", 0, "Base price in USD cents")
	nodesPricesCmd.Flags().Int64("storage", 0, "Storage price in USD cents")
	nodesPricesCmd.Flags().Int64("traffic", 0, "Traffic price in USD cents")
	nodesPricesCmd.Flags().StringP("output", "o", "table", "Output format: table or json")

	nodesRebootCmd.Flags().StringP("output", "o", "table", "Output format: table or json")

	nodesCmd.AddCommand(nodesListCmd)
	nodesCmd.AddCommand(nodesShowCmd)
	nodesCmd.AddCommand(nodesIdentCmd)
	nodesCmd.AddCommand(nodesLogsCmd)
	nodesCmd.AddCommand(nodesPricesCmd)
	nodesCmd.AddCommand(nodesRebootCmd)
}
