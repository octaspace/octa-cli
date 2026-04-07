package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/octaspace/octa/internal/api"
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
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		client := api.NewClient(cfg.APIKey)

		format := outputFormat

		if format == "json" {
			raw, err := client.ListNodesRaw()
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}

			var buf interface{}
			if err := json.Unmarshal(raw, &buf); err != nil {
				fmt.Fprintln(os.Stderr, "could not parse JSON:", err)
				os.Exit(1)
			}

			pretty, err := json.MarshalIndent(buf, "", "  ")
			if err != nil {
				fmt.Fprintln(os.Stderr, "could not format JSON:", err)
				os.Exit(1)
			}

			fmt.Println(string(pretty))
			return nil
		}

		nodes, err := client.ListNodes()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		return ui.RenderNodesTable(nodes)
	},
}

func init() {
	nodesListCmd.Flags().StringVarP(&outputFormat, "output", "o", "table", "Output format: table or json")
	nodesCmd.AddCommand(nodesListCmd)
}
