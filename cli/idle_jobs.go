package cli

import (
	"encoding/json"
	"fmt"

	"github.com/octaspace/octa/internal/client"
	"github.com/octaspace/octa/internal/config"
	"github.com/spf13/cobra"
)

var idleJobsCmd = &cobra.Command{
	Use:   "idle-jobs",
	Short: "Inspect idle jobs",
}

var idleJobsShowCmd = &cobra.Command{
	Use:   "show <node-id> <job-id>",
	Short: "Show the status of an idle job",
	Args:  exactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		nodeID, jobID, err := parseIdleJobIDs(args)
		if err != nil {
			return err
		}

		cfg, err := config.Load()
		if err != nil {
			return err
		}

		c := client.New(cfg)
		job, err := c.IdleJobs.Find(cmd.Context(), nodeID, jobID)
		if err != nil {
			return client.Friendly(err)
		}

		out, err := json.MarshalIndent(job, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(out))
		return nil
	},
}

var idleJobsLogsCmd = &cobra.Command{
	Use:   "logs <node-id> <job-id>",
	Short: "Fetch the logs of an idle job",
	Args:  exactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		nodeID, jobID, err := parseIdleJobIDs(args)
		if err != nil {
			return err
		}

		cfg, err := config.Load()
		if err != nil {
			return err
		}

		c := client.New(cfg)
		data, err := c.IdleJobs.Logs(cmd.Context(), nodeID, jobID)
		if err != nil {
			return client.Friendly(err)
		}

		out, _ := cmd.Flags().GetString("out")
		if out == "" {
			fmt.Print(string(data))
			return nil
		}
		return writeDownload(out, data)
	},
}

func parseIdleJobIDs(args []string) (nodeID, jobID int64, err error) {
	nodeID, err = parseNodeID(args[0])
	if err != nil {
		return 0, 0, err
	}
	jobID, err = parseNodeID(args[1])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid job ID %q", args[1])
	}
	return nodeID, jobID, nil
}

func init() {
	idleJobsLogsCmd.Flags().StringP("out", "o", "", "Output file path (default stdout)")
	idleJobsCmd.AddCommand(idleJobsShowCmd)
	idleJobsCmd.AddCommand(idleJobsLogsCmd)
}
