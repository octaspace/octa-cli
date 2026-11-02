package cli

import (
	"fmt"
	"sort"

	"github.com/octaspace/octa/internal/client"
	"github.com/octaspace/octa/internal/config"
	"github.com/octaspace/octa/internal/ui"
	"github.com/spf13/cobra"
)

var renderCmd = &cobra.Command{
	Use:   "render",
	Short: "List render nodes and rent them",
	Long: `List the nodes available for render: an NVIDIA GPU with a Blender score.

Works without an API token, but the anonymous listing is refreshed only
hourly. Render in the new API is still being migrated: its behaviour may
change.`,
	Args: noArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := listingClient()
		if err != nil {
			return err
		}

		format, _ := cmd.Flags().GetString("output")
		if format == "json" {
			out, err := c.Render.ListRaw(cmd.Context())
			if err != nil {
				return client.Friendly(err)
			}
			fmt.Println(string(out))
			return nil
		}
		machines, err := c.Render.List(cmd.Context())
		if err != nil {
			return client.Friendly(err)
		}
		if len(machines) == 0 {
			fmt.Println("No render nodes available.")
			return nil
		}

		sort.SliceStable(machines, func(i, j int) bool {
			return machines[i].BlenderScore > machines[j].BlenderScore
		})
		return ui.RenderRenderTable(machines)
	},
}

var renderStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Rent render nodes",
	Long: `Rent a render node from 'octa render'. Repeat --node to rent several in one
batch request; each request succeeds or fails on its own. A session is billed
by the hour (the base price) until it is stopped with 'octa sessions stop'.`,
	Example: `  octa render start --node 7KQ4-M2XD-4 --wait
  octa render start --node 7KQ4-M2XD-4 --node M95S-WZG8-7`,
	Args: noArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		raw, _ := cmd.Flags().GetStringArray("node")
		nodes := nonEmpty(raw)
		if len(nodes) == 0 {
			return fmt.Errorf("--node required")
		}

		cfg, err := config.Load()
		if err != nil {
			return err
		}
		c := client.New(cfg)

		if len(nodes) > 1 {
			return startRenderBatch(cmd, c, nodes)
		}
		uuid, err := c.Render.Start(cmd.Context(), nodes[0])
		if err != nil {
			return client.Friendly(err)
		}
		return reportStarted(cmd, c, uuid)
	},
}

func init() {
	renderCmd.Flags().StringP("output", "o", "table", "Output format: table or json")
	renderStartCmd.Flags().StringP("output", "o", "table", "Output format: table or json")
	renderStartCmd.Flags().StringArray("node", nil, "Node number from 'octa render', e.g. 7KQ4-M2XD-4; repeat for a batch (required)")
	addWaitFlags(renderStartCmd)
	renderCmd.AddCommand(renderStartCmd)
}
