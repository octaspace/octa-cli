package cli

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/charmbracelet/lipgloss"
	octaspace "github.com/octaspace/go-sdk"
	"github.com/octaspace/octa/internal/client"
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
			return err
		}

		c := client.New(cfg)
		format, _ := cmd.Flags().GetString("output")
		if format == "json" {
			out, err := c.Services.MR.ListRaw(cmd.Context(), nil)
			if err != nil {
				return client.Friendly(err)
			}
			fmt.Println(string(out))
			return nil
		}
		machines, err := c.Services.MR.List(cmd.Context(), nil)
		if err != nil {
			return client.Friendly(err)
		}

		sortByGPUCount(machines)
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
			return err
		}

		machines, err := client.New(cfg).Services.MR.List(cmd.Context(), nil)
		if err != nil {
			return client.Friendly(err)
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

		sortByGPUCount(filtered)

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
		ports, _ := cmd.Flags().GetIntSlice("ports")
		httpPorts, _ := cmd.Flags().GetIntSlice("http-ports")
		startCommand, _ := cmd.Flags().GetString("start-command")
		entrypoint, _ := cmd.Flags().GetString("entrypoint")

		if app == "" && image == "" {
			return errors.New("--app or --image is required")
		}
		if nodeID == 0 {
			return errors.New("--node is required")
		}

		cfg, err := config.Load()
		if err != nil {
			return err
		}

		c := client.New(cfg)

		var appTemplate *octaspace.App
		if app != "" {
			apps, err := c.Apps.List(cmd.Context())
			if err != nil {
				return client.Friendly(err)
			}
			for i := range apps {
				if apps[i].UUID == app {
					appTemplate = &apps[i]
					break
				}
			}
			if appTemplate == nil {
				return fmt.Errorf("app %q not found", app)
			}
		}

		params, err := buildDeployParams(deployParamsInput{
			NodeID: nodeID, AppUUID: app, Template: appTemplate,
			Image: image, ImageSet: cmd.Flags().Changed("image"),
			DiskSize: diskSize, DiskSet: cmd.Flags().Changed("disk"),
			Envs: envsStr, EnvsSet: cmd.Flags().Changed("envs"),
			Ports: ports, PortsSet: cmd.Flags().Changed("ports"),
			HTTPPorts: httpPorts, HTTPPortsSet: cmd.Flags().Changed("http-ports"),
			StartCommand: startCommand, StartCommandSet: cmd.Flags().Changed("start-command"),
			Entrypoint: entrypoint,
		})
		if err != nil {
			return err
		}

		resp, err := c.Services.MR.Create(cmd.Context(), params)
		if err != nil {
			return client.Friendly(err)
		}

		uuids := resp.UUIDs()
		if len(uuids) == 0 {
			reason := "deploy request rejected"
			for _, r := range resp.Results {
				if r.Reason != "" {
					reason = r.Reason
					break
				}
				if r.Message != "" {
					reason = r.Message
					break
				}
			}
			return errors.New(reason)
		}
		for _, u := range uuids {
			fmt.Printf("Session UUID: %s\n", u)
		}
		return nil
	},
}

var computeAppsCmd = &cobra.Command{
	Use:   "apps",
	Short: "List available applications",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		c := client.New(cfg)
		format, _ := cmd.Flags().GetString("output")
		if format == "json" {
			out, err := c.Apps.ListRaw(cmd.Context())
			if err != nil {
				return client.Friendly(err)
			}
			fmt.Println(string(out))
			return nil
		}
		apps, err := c.Apps.List(cmd.Context())
		if err != nil {
			return client.Friendly(err)
		}

		return ui.RenderAppsTable(apps)
	},
}

type deployParamsInput struct {
	NodeID          int64
	AppUUID         string
	Template        *octaspace.App
	Image           string
	ImageSet        bool
	DiskSize        int
	DiskSet         bool
	Envs            string
	EnvsSet         bool
	Ports           []int
	PortsSet        bool
	HTTPPorts       []int
	HTTPPortsSet    bool
	StartCommand    string
	StartCommandSet bool
	Entrypoint      string
}

func buildDeployParams(input deployParamsInput) (*octaspace.MachineRentalCreateParams, error) {
	params := &octaspace.MachineRentalCreateParams{
		NodeID: input.NodeID, App: input.AppUUID, Image: input.Image,
		DiskSize: input.DiskSize, Ports: input.Ports, HTTPPorts: input.HTTPPorts,
		StartCommand: input.StartCommand, Entrypoint: input.Entrypoint,
	}
	if input.Template != nil {
		if !input.ImageSet {
			params.Image = input.Template.Image
		}
		if !input.DiskSet {
			if value, ok := input.Template.Extra["min_disk_size"].(float64); ok {
				params.DiskSize = int(value)
			}
		}
		if !input.PortsSet {
			params.Ports = append([]int(nil), input.Template.Ports...)
		}
		if !input.HTTPPortsSet {
			params.HTTPPorts = append([]int(nil), input.Template.HTTPPorts...)
		}
		if !input.StartCommandSet {
			params.StartCommand = input.Template.StartCommand
		}
		params.Envs = normalizeAppEnvs(input.Template.Envs)
	}
	if input.EnvsSet {
		userEnvs, err := parseDeployEnvs(input.Envs)
		if err != nil {
			return nil, err
		}
		if params.Envs == nil {
			params.Envs = map[string]string{}
		}
		for key, value := range userEnvs {
			params.Envs[key] = value
		}
	}
	return params, nil
}

func normalizeAppEnvs(values map[string]any) map[string]string {
	if values == nil {
		return nil
	}
	result := make(map[string]string, len(values))
	for key, value := range values {
		if value == nil {
			result[key] = ""
			continue
		}
		result[key] = fmt.Sprint(value)
	}
	return result
}

func parseDeployEnvs(value string) (map[string]string, error) {
	result := map[string]string{}
	if value == "" {
		return result, nil
	}
	for _, pair := range strings.Split(value, ",") {
		parts := strings.SplitN(pair, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid env format %q, expected KEY=VALUE", pair)
		}
		result[parts[0]] = parts[1]
	}
	return result, nil
}

var computeLogsCmd = &cobra.Command{
	Use:   "logs <uuid>",
	Short: "Show system and container logs for a session",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		c := client.New(cfg)
		sessions, err := c.Sessions.List(cmd.Context(), nil)
		if err != nil {
			return client.Friendly(err)
		}

		session, err := client.MatchSession(sessions, args[0])
		if err != nil {
			return err
		}

		logs, err := c.Services.Session(session.UUID).Logs(cmd.Context(), nil)
		if err != nil {
			return client.Friendly(err)
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
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		sessions, err := client.New(cfg).Sessions.List(cmd.Context(), nil)
		if err != nil {
			return client.Friendly(err)
		}

		session, err := client.MatchSession(sessions, args[0])
		if err != nil {
			return err
		}

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
			return errors.New("no SSH endpoint available for this session")
		}

		sshPath, err := exec.LookPath("ssh")
		if err != nil {
			return errors.New("ssh not found in PATH")
		}

		return syscall.Exec(sshPath, []string{
			"ssh", "-p", fmt.Sprintf("%d", port), fmt.Sprintf("root@%s", host),
		}, os.Environ())
	},
}

// sortByGPUCount orders machines by descending GPU count (most GPUs first).
func sortByGPUCount(machines []octaspace.MachineRental) {
	sort.SliceStable(machines, func(i, j int) bool {
		return len(machines[i].GPUs) > len(machines[j].GPUs)
	})
}

func init() {
	computeCmd.Flags().StringP("output", "o", "table", "Output format: table or json")
	computeAppsCmd.Flags().StringP("output", "o", "table", "Output format: table or json")
	computeDeployCmd.Flags().String("app", "", "Application UUID")
	computeDeployCmd.Flags().Int64("node", 0, "Node ID")
	computeDeployCmd.Flags().String("image", "", "Docker image to run (optional)")
	computeDeployCmd.Flags().Int("disk", 0, "Disk size in GB (default: app's min_disk_size)")
	computeDeployCmd.Flags().String("envs", "", "Environment variables in KEY=VALUE format, comma-separated (e.g. ENV1=VAL1,ENV2=VAL2)")
	computeDeployCmd.Flags().IntSlice("ports", nil, "TCP/UDP ports to expose (overrides app defaults)")
	computeDeployCmd.Flags().IntSlice("http-ports", nil, "HTTP ports to expose (overrides app defaults)")
	computeDeployCmd.Flags().String("start-command", "", "Container start command (overrides app default)")
	computeDeployCmd.Flags().String("entrypoint", "", "Container entrypoint")
	computeConnectCmd.Flags().Bool("proxy", false, "Force connection via proxy instead of direct SSH")
	computeCmd.AddCommand(computeSearchCmd)
	computeCmd.AddCommand(computeAppsCmd)
	computeCmd.AddCommand(computeDeployCmd)
	computeCmd.AddCommand(computeLogsCmd)
	computeCmd.AddCommand(computeConnectCmd)
}
