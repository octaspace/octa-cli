package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/octaspace/octa/internal/api"
	"github.com/octaspace/octa/internal/config"
	"github.com/octaspace/octa/internal/ui"
	"github.com/spf13/cobra"
)

var computeCmd = &cobra.Command{
	Use:   "compute",
	Short: "List available machines for rent",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		client := api.NewClient(cfg.APIKey)

		format, _ := cmd.Flags().GetString("output")
		if format == "json" {
			raw, err := client.ListMachinesForRentRaw()
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			var buf interface{}
			json.Unmarshal(raw, &buf)
			pretty, _ := json.MarshalIndent(buf, "", "  ")
			fmt.Println(string(pretty))
			return nil
		}

		machines, err := client.ListMachinesForRent()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		sort.SliceStable(machines, func(i, j int) bool {
			return len(machines[i].GPUs) > len(machines[j].GPUs)
		})

		return ui.RenderComputeTable(machines)
	},
}

var computeSearchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search available machines by CPU or GPU model",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		query := strings.ToLower(args[0])

		cfg, err := config.Load()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		machines, err := api.NewClient(cfg.APIKey).ListMachinesForRent()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		filtered := machines[:0]
		for _, m := range machines {
			if strings.Contains(strings.ToLower(m.CPUModelName), query) ||
				strings.Contains(strings.ToLower(m.Country), query) {
				filtered = append(filtered, m)
				continue
			}
			for _, g := range m.GPUs {
				if strings.Contains(strings.ToLower(g.Model), query) {
					filtered = append(filtered, m)
					break
				}
			}
		}

		sort.SliceStable(filtered, func(i, j int) bool {
			return len(filtered[i].GPUs) > len(filtered[j].GPUs)
		})

		if len(filtered) == 0 {
			fmt.Println("No machines found.")
			return nil
		}

		return ui.RenderComputeTable(filtered)
	},
}

var computeDeployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "Deploy an application on a node",
	RunE: func(cmd *cobra.Command, args []string) error {
		app, _ := cmd.Flags().GetString("app")
		nodeID, _ := cmd.Flags().GetInt64("node")
		image, _ := cmd.Flags().GetString("image")
		diskSize, _ := cmd.Flags().GetInt("disk")
		envsStr, _ := cmd.Flags().GetString("envs")

		if app == "" && image == "" {
			fmt.Fprintln(os.Stderr, "error: --app or --image is required")
			os.Exit(1)
		}
		if nodeID == 0 {
			fmt.Fprintln(os.Stderr, "error: --node is required")
			os.Exit(1)
		}

		cfg, err := config.Load()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		client := api.NewClient(cfg.APIKey)

		if (image == "" || diskSize == 0) && app != "" {
			apps, err := client.ListApps()
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			var found *api.App
			for i := range apps {
				if apps[i].UUID == app {
					found = &apps[i]
					break
				}
			}
			if found == nil {
				fmt.Fprintf(os.Stderr, "error: app %q not found\n", app)
				os.Exit(1)
			}
			if image == "" {
				image = found.Image
			}
			if diskSize == 0 {
				diskSize = found.Extra.MinDiskSize
			}
		}

		var envs map[string]string
		if envsStr != "" {
			envs = make(map[string]string)
			for _, pair := range strings.Split(envsStr, ",") {
				parts := strings.SplitN(pair, "=", 2)
				if len(parts) != 2 {
					fmt.Fprintf(os.Stderr, "error: invalid env format %q, expected KEY=VALUE\n", pair)
					os.Exit(1)
				}
				envs[parts[0]] = parts[1]
			}
		}

		resp, err := client.DeployMachine(api.DeployRequest{
			App:      app,
			NodeID:   nodeID,
			Image:    image,
			DiskSize: diskSize,
			Envs:     envs,
		})
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		fmt.Printf("Session UUID: %s\n", resp.UUID)
		return nil
	},
}

var computeAppsCmd = &cobra.Command{
	Use:   "apps",
	Short: "List available applications",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		client := api.NewClient(cfg.APIKey)

		format, _ := cmd.Flags().GetString("output")
		if format == "json" {
			raw, err := client.ListAppsRaw()
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			var buf interface{}
			json.Unmarshal(raw, &buf)
			pretty, _ := json.MarshalIndent(buf, "", "  ")
			fmt.Println(string(pretty))
			return nil
		}

		apps, err := client.ListApps()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		return ui.RenderAppsTable(apps)
	},
}

var computeLogsCmd = &cobra.Command{
	Use:   "logs <uuid>",
	Short: "Show system and container logs for a session",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		input := args[0]

		cfg, err := config.Load()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		client := api.NewClient(cfg.APIKey)

		sessions, err := client.ListSessions()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		var matched []api.Session
		for _, s := range sessions {
			if s.UUID == input || strings.HasPrefix(s.UUID, input) {
				matched = append(matched, s)
			}
		}

		switch len(matched) {
		case 0:
			fmt.Fprintf(os.Stderr, "no session found matching %q\n", input)
			os.Exit(1)
		default:
			if len(matched) > 1 {
				fmt.Fprintf(os.Stderr, "ambiguous UUID %q matches %d sessions:\n", input, len(matched))
				for _, s := range matched {
					fmt.Fprintf(os.Stderr, "  %s\n", s.UUID)
				}
				os.Exit(1)
			}
		}

		logs, err := client.GetSessionLogs(matched[0].UUID)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		header := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#cc1b99"))

		fmt.Println(header.Render("=== System ==="))
		for _, e := range logs.System {
			fmt.Printf("%s  %s\n", time.UnixMilli(e.TS).Format("2006-01-02 15:04:05"), e.Msg)
		}
		fmt.Println(header.Render("=== Container ==="))
		fmt.Println(logs.Container)

		return nil
	},
}

var computeConnectCmd = &cobra.Command{
	Use:   "connect <uuid>",
	Short: "Connect to a session via SSH",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		input := args[0]

		cfg, err := config.Load()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		sessions, err := api.NewClient(cfg.APIKey).ListSessions()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		var matched []api.Session
		for _, s := range sessions {
			if s.UUID == input || strings.HasPrefix(s.UUID, input) {
				matched = append(matched, s)
			}
		}

		switch len(matched) {
		case 0:
			fmt.Fprintf(os.Stderr, "no session found matching %q\n", input)
			os.Exit(1)
		default:
			if len(matched) > 1 {
				fmt.Fprintf(os.Stderr, "ambiguous UUID %q matches %d sessions:\n", input, len(matched))
				for _, s := range matched {
					fmt.Fprintf(os.Stderr, "  %s\n", s.UUID)
				}
				os.Exit(1)
			}
		}

		session := matched[0]

		forceProxy, _ := cmd.Flags().GetBool("proxy")

		var host string
		var port int
		if !forceProxy && session.SSHDirect.Host != "" && session.SSHDirect.Port != 0 {
			host = session.SSHDirect.Host
			port = session.SSHDirect.Port
		} else if session.SSHProxy.Host != "" && session.SSHProxy.Port != 0 {
			host = session.SSHProxy.Host
			port = session.SSHProxy.Port
		} else {
			fmt.Fprintln(os.Stderr, "error: no SSH endpoint available for this session")
			os.Exit(1)
		}

		sshPath, err := exec.LookPath("ssh")
		if err != nil {
			fmt.Fprintln(os.Stderr, "error: ssh not found in PATH")
			os.Exit(1)
		}

		return syscall.Exec(sshPath, []string{
			"ssh", "-p", fmt.Sprintf("%d", port), fmt.Sprintf("root@%s", host),
		}, os.Environ())
	},
}

func init() {
	computeCmd.Flags().StringP("output", "o", "table", "Output format: table or json")
	computeAppsCmd.Flags().StringP("output", "o", "table", "Output format: table or json")
	computeDeployCmd.Flags().String("app", "", "Application UUID")
	computeDeployCmd.Flags().Int64("node", 0, "Node ID")
	computeDeployCmd.Flags().String("image", "", "Docker image to run (optional)")
	computeDeployCmd.Flags().Int("disk", 0, "Disk size in GB (default: app's min_disk_size)")
	computeDeployCmd.Flags().String("envs", "", "Environment variables in KEY=VALUE format, comma-separated (e.g. ENV1=VAL1,ENV2=VAL2)")
	computeConnectCmd.Flags().Bool("proxy", false, "Force connection via proxy instead of direct SSH")
	computeCmd.AddCommand(computeSearchCmd)
	computeCmd.AddCommand(computeAppsCmd)
	computeCmd.AddCommand(computeDeployCmd)
	computeCmd.AddCommand(computeLogsCmd)
	computeCmd.AddCommand(computeConnectCmd)
}
