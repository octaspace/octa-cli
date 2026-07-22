package cli

import (
	"fmt"

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

		c := client.New(cfg)
		format, _ := cmd.Flags().GetString("output")
		if format == "json" {
			out, err := c.Nodes.ListRaw(cmd.Context(), nil)
			if err != nil {
				return client.Friendly(err)
			}
			fmt.Println(string(out))
			return nil
		}
		nodes, err := c.Nodes.List(cmd.Context(), nil)
		if err != nil {
			return client.Friendly(err)
		}

		return ui.RenderNodesTable(nodes)
	},
}

func init() {
	nodesListCmd.Flags().StringP("output", "o", "table", "Output format: table or json")
	nodesCmd.AddCommand(nodesListCmd)
}
