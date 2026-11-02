package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
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

const defaultWaitTimeout = 15 * time.Minute

var computeCmd = &cobra.Command{
	Use:   "compute",
	Short: "List machines for rent and manage compute sessions",
	Long: `List the machines available for rent.

Works without an API token, but the anonymous listing is refreshed only
hourly; with a token it is at most 10 seconds old.`,
	Args: noArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := listingClient()
		if err != nil {
			return err
		}

		format, _ := cmd.Flags().GetString("output")
		if format == "json" {
			out, err := c.Compute.ListRaw(cmd.Context())
			if err != nil {
				return client.Friendly(err)
			}
			fmt.Println(string(out))
			return nil
		}
		machines, err := c.Compute.List(cmd.Context())
		if err != nil {
			return client.Friendly(err)
		}
		if len(machines) == 0 {
			fmt.Println("No machines available.")
			return nil
		}

		sortByGPUCount(machines)
		return ui.RenderComputeTable(machines)
	},
}

var computeSearchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search machines by CPU or GPU model, country or city",
	Args:  exactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := listingClient()
		if err != nil {
			return err
		}
		machines, err := c.Compute.List(cmd.Context())
		if err != nil {
			return client.Friendly(err)
		}

		filtered := filterMachines(machines, args[0])
		sortByGPUCount(filtered)

		format, _ := cmd.Flags().GetString("output")
		if format == "json" {
			return printJSON(filtered)
		}
		if len(filtered) == 0 {
			fmt.Println("No machines found.")
			return nil
		}
		return ui.RenderComputeTable(filtered)
	},
}

var computeStartCmd = &cobra.Command{
	Use:     "start",
	Aliases: []string{"deploy"},
	Short:   "Start compute sessions on machines",
	Long: `Start a compute session on a machine from 'octa compute'.

--node and --image are required. Repeat --node to start the same session on
several machines in one batch request, or pass --from-file with a JSON array of
requests that differ. In a batch each request succeeds or fails on its own.

The session is billed from the start; one stopped before the machine's minimum
rent is charged base and storage for the whole minimum. Starting is
asynchronous: use --wait to follow the progress until the session is ready.
Starts are never retried automatically.`,
	Example: `  octa compute start --node 7KQ4-M2XD-4 --image ubuntu:24.04 --ssh --wait
  octa compute start --node 7KQ4-M2XD-4 --image nginx --http-ports 80 -e MODE=prod
  octa compute start --node 7KQ4-M2XD-4 --image alpine --entrypoint "sh -c" --args "'sleep infinity'"
  octa compute start --node 7KQ4-M2XD-4 --node M95S-WZG8-7 --image ubuntu:24.04
  octa compute start --from-file batch.json --wait`,
	Args: noArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		items, err := computeStartItems(cmd)
		if err != nil {
			return err
		}

		cfg, err := config.Load()
		if err != nil {
			return err
		}
		c := client.New(cfg)

		if len(items) == 1 && !cmd.Flags().Changed("from-file") {
			uuid, err := c.Compute.Start(cmd.Context(), &items[0].ComputeStartParams)
			if err != nil {
				return client.Friendly(err)
			}
			return reportStarted(cmd, c, uuid)
		}

		results, err := c.Compute.StartBatch(cmd.Context(), items)
		if err != nil {
			return client.Friendly(err)
		}
		nodes := make(map[string]string, len(items))
		for _, it := range items {
			nodes[it.ID] = it.NodeNumber
		}
		return reportBatch(cmd, c, results, nodes)
	},
}

// computeStartItems builds the requests of 'compute start' from the flags or
// from --from-file.
func computeStartItems(cmd *cobra.Command) ([]octaspace.ComputeBatchItem, error) {
	file, _ := cmd.Flags().GetString("from-file")
	if file != "" {
		var conflicting []string
		for _, name := range append([]string{"node", "image", "app"}, sessionParamFlags...) {
			if cmd.Flags().Changed(name) {
				conflicting = append(conflicting, "--"+name)
			}
		}
		if len(conflicting) > 0 {
			return nil, fmt.Errorf("--from-file cannot be combined with %s: put them in the file", strings.Join(conflicting, ", "))
		}
		return loadComputeBatchFile(file)
	}

	nodes, _ := cmd.Flags().GetStringArray("node")
	image, _ := cmd.Flags().GetString("image")
	app, _ := cmd.Flags().GetString("app")
	image, app = strings.TrimSpace(image), strings.TrimSpace(app)

	var missing []string
	if len(nonEmpty(nodes)) == 0 {
		missing = append(missing, "--node")
	}
	if image == "" {
		missing = append(missing, "--image")
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("%s required (or --from-file)", strings.Join(missing, " and "))
	}

	params, err := buildSessionParams(sessionParamsFromFlags(cmd))
	if err != nil {
		return nil, err
	}
	var items []octaspace.ComputeBatchItem
	for i, node := range nonEmpty(nodes) {
		items = append(items, octaspace.ComputeBatchItem{
			ID:                 strconv.Itoa(i + 1),
			ComputeStartParams: *params.startParams(node, image, app),
		})
	}
	return items, nil
}

var computeRestartCmd = &cobra.Command{
	Use:   "restart <uuid>",
	Short: "Restart the container of a compute session",
	Long: `Recreate the container of a running compute session on the same node. The
rent, the session UUID and its prices stay the same.

Without parameter flags the container restarts with the parameters the session
was started with.

Parameter flags REPLACE ALL of the session's parameters, they are not merged.
Every parameter left out (or given empty) gets its fixed default, not the
session's current value:

  --disk                 1 GB (so --disk is required with --replace)
  --entrypoint, --args   the image's own ENTRYPOINT and CMD
  --startup-script       none
  -e/--env               no variables
  --ports, --http-ports  no published ports
  --ssh, --fm            the app's setting: off for a plain image

The API does not report the entrypoint, args, startup script or disk size a
session was started with, and shows its env, ports and SSH/file manager only as
the session's current state, so the CLI cannot fill them in. Pass the full set
and confirm with --replace; the CLI warns about the env, ports, SSH and file
manager the session has now that the new set would drop.

The restart is asynchronous: the API accepts it (202) and the session stops if
the new parameters cannot be set up or its command exits. A session can be
restarted once in 5 minutes and 6 times an hour, and not while it is still
starting or restarting.`,
	Example: `  octa compute restart 2a63b54f
  octa compute restart 2a63b54f --replace --disk 20 \
    --entrypoint "sh -c" --args "'sleep infinity'" --ssh -e MODE=prod`,
	Args: exactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		params, err := buildSessionParams(sessionParamsFromFlags(cmd))
		if err != nil {
			return err
		}
		restart := params.restartParams()
		replace, _ := cmd.Flags().GetBool("replace")
		if restart != nil && !replace {
			return errors.New("parameter flags replace ALL of the session's parameters: anything not given " +
				"(entrypoint, args, SSH, ports, env, ...) is reset to the default, which can stop the session.\n" +
				"Pass the full set of parameters and add --replace to confirm, or drop the flags to restart with the original parameters")
		}
		if restart == nil && replace {
			return errors.New("--replace needs the parameter flags to replace the session's parameters with")
		}
		if restart != nil && !cmd.Flags().Changed("disk") {
			return errors.New("--disk is required with --replace: a restart without it shrinks the disk to 1 GB")
		}

		cfg, err := config.Load()
		if err != nil {
			return err
		}
		c := client.New(cfg)

		session, err := findSession(cmd.Context(), c, args[0])
		if err != nil {
			return err
		}
		if !session.IsCompute() {
			return fmt.Errorf("session %s is a %s session: only compute sessions can be restarted", session.UUID, session.Service)
		}

		if restart != nil {
			for _, w := range restartDropWarnings(session, restart) {
				fmt.Fprintln(os.Stderr, "Warning: "+w)
			}
		}

		format, _ := cmd.Flags().GetString("output")
		if format != "json" {
			what := "with its original parameters"
			if restart != nil {
				what = "with the new parameters"
			}
			fmt.Fprintf(os.Stderr, "Restarting session %s (%s on %s) %s.\n", session.UUID, orDash(session.AppName), orDash(session.NodeNumber), what)
		}

		uuid, err := c.Compute.Restart(cmd.Context(), session.UUID, restart)
		if err != nil {
			return client.Friendly(err)
		}
		if wait, _ := cmd.Flags().GetBool("wait"); wait {
			// The session was ready before the restart: wait until it leaves
			// that state, or --wait could report the old container as ready.
			waitLeftReady(cmd.Context(), c, uuid, restartLeaveReadyTimeout)
		}
		return reportStarted(cmd, c, uuid)
	},
}

var computeLogsCmd = &cobra.Command{
	Use:   "logs <uuid>",
	Short: "Show the system and container logs of a compute session",
	Long: `Show the system events and the latest container output of a running compute
session. The logs of a stopped session are no longer available.`,
	Args: exactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		tail, _ := cmd.Flags().GetInt("tail")
		if cmd.Flags().Changed("tail") && (tail < 1 || tail > 1000) {
			return errors.New("--tail must be between 1 and 1000")
		}

		cfg, err := config.Load()
		if err != nil {
			return err
		}
		c := client.New(cfg)

		session, err := findSession(cmd.Context(), c, args[0])
		if err != nil {
			return err
		}

		logs, err := c.Sessions.Logs(cmd.Context(), session.UUID, &octaspace.LogsParams{Tail: tail})
		if err != nil {
			return client.Friendly(err)
		}

		format, _ := cmd.Flags().GetString("output")
		if format == "json" {
			return printJSON(logs)
		}

		header := lipgloss.NewStyle().Bold(true)
		fmt.Println(header.Render("=== System ==="))
		for _, e := range logs.System {
			ts := "                   "
			if t := e.Time(); !t.IsZero() {
				ts = t.Local().Format(time.DateTime)
			}
			fmt.Printf("%s  %s\n", ts, e.Msg)
		}
		fmt.Println(header.Render("=== Container ==="))
		if logs.Container == nil {
			fmt.Println("(container output unavailable: the node cannot be reached)")
			return nil
		}
		if logs.ContainerTruncated {
			fmt.Println("(truncated to the last 1 MiB)")
		}
		fmt.Println(strings.ReplaceAll(*logs.Container, "\r\n", "\n"))
		return nil
	},
}

var computeConnectCmd = &cobra.Command{
	Use:   "connect <uuid>",
	Short: "Connect to a compute session via SSH",
	Long: `Connect to a compute session via SSH. The session must have been started
with --ssh (or an app that enables it).`,
	Args: exactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		session, err := findSession(cmd.Context(), client.New(cfg), args[0])
		if err != nil {
			return err
		}

		forceProxy, _ := cmd.Flags().GetBool("proxy")
		endpoint := session.SSH.Direct
		if forceProxy || !validEndpoint(endpoint) {
			endpoint = session.SSH.Proxy
		}
		if !validEndpoint(endpoint) {
			if session.SSH.Web != "" {
				return fmt.Errorf("no SSH endpoint available for this session; web terminal: %s", session.SSH.Web)
			}
			return errors.New("no SSH endpoint available for this session: start it with --ssh")
		}

		sshPath, err := exec.LookPath("ssh")
		if err != nil {
			return errors.New("ssh not found in PATH")
		}

		argv, err := sshArgs(endpoint)
		if err != nil {
			return err
		}
		return syscall.Exec(sshPath, argv, os.Environ())
	},
}

// sshUserRe matches the user names ssh is given. The user comes from the
// image's USER, which anyone building an image controls.
var sshUserRe = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9._-]*$`)

// sshHostRe matches a host name or an IPv4/IPv6 address.
var sshHostRe = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9.:-]*$`)

// sshArgs builds the ssh command line for an endpoint reported by the API.
// The user and host are validated and passed as separate arguments after
// "--", so a value starting with "-" can never be read as an ssh option
// (e.g. -oProxyCommand=..., which would run a local command).
func sshArgs(e *octaspace.SSHEndpoint) ([]string, error) {
	user := e.User
	if user == "" {
		user = "root"
	}
	if !sshUserRe.MatchString(user) {
		return nil, fmt.Errorf("refusing to connect: unexpected SSH user %q reported by the API", user)
	}
	if !sshHostRe.MatchString(e.Host) {
		return nil, fmt.Errorf("refusing to connect: unexpected SSH host %q reported by the API", e.Host)
	}
	if e.Port < 1 || e.Port > 65535 {
		return nil, fmt.Errorf("refusing to connect: unexpected SSH port %d reported by the API", e.Port)
	}
	return []string{"ssh", "-p", strconv.Itoa(e.Port), "-l", user, "--", e.Host}, nil
}

func validEndpoint(e *octaspace.SSHEndpoint) bool {
	return e != nil && e.Host != "" && e.Port != 0
}

// listingClient builds a client for the listings, which also work without a
// token: then it warns that the anonymous listing may be up to an hour old.
func listingClient() (*octaspace.Client, error) {
	cfg, err := config.LoadOptional()
	if err != nil {
		return nil, err
	}
	if cfg.APIKey != "" && !config.IsAPIKey(cfg.APIKey) {
		// Never send a legacy token: list anonymously instead.
		fmt.Fprintf(os.Stderr, "Note: %v.\n", config.ErrLegacyToken)
		cfg.APIKey = ""
	}
	if cfg.APIKey == "" {
		fmt.Fprintln(os.Stderr, "Note: no API key configured; the anonymous listing may be up to an hour old. Run 'octa auth <key>' for a fresh one.")
	}
	return client.New(cfg), nil
}

// findSession resolves a full or partial UUID among the running sessions.
func findSession(ctx context.Context, c *octaspace.Client, query string) (octaspace.Session, error) {
	sessions, err := c.Sessions.List(ctx)
	if err != nil {
		return octaspace.Session{}, client.Friendly(err)
	}
	return client.MatchSession(sessions, query)
}

// addWaitFlags registers --wait and --wait-timeout on a command that starts
// a session.
func addWaitFlags(cmd *cobra.Command) {
	cmd.Flags().Bool("wait", false, "Wait until the session is ready, showing its progress")
	cmd.Flags().Duration("wait-timeout", defaultWaitTimeout, "How long --wait waits for each session")
}

// reportStarted prints the UUID of a started (or restarted) session and, with
// --wait, follows it until it is ready.
func reportStarted(cmd *cobra.Command, c *octaspace.Client, uuid string) error {
	format, _ := cmd.Flags().GetString("output")
	wait, _ := cmd.Flags().GetBool("wait")
	jsonOut := format == "json"

	if !wait {
		if jsonOut {
			return printJSON(map[string]any{"status": "ok", "uuid": uuid})
		}
		fmt.Printf("Session UUID: %s\n", uuid)
		fmt.Printf("Follow it with 'octa sessions info %s'.\n", uuid)
		return nil
	}

	if !jsonOut {
		fmt.Printf("Session UUID: %s\n", uuid)
	}
	timeout, _ := cmd.Flags().GetDuration("wait-timeout")
	session, err := waitReady(cmd.Context(), c, uuid, timeout, !jsonOut)
	if err != nil {
		return err
	}
	if jsonOut {
		return printJSON(session)
	}
	printSessionAccess(*session)
	return nil
}

// restartDropWarnings lists what a session has now that the replacement
// parameters would drop. Only env, ports, SSH and the file manager are
// visible in the session; the rest cannot be checked.
func restartDropWarnings(s octaspace.Session, p *octaspace.ComputeRestartParams) []string {
	var w []string
	if len(s.Envs) > 0 && len(p.Envs) == 0 {
		w = append(w, fmt.Sprintf("the session's %d env variable(s) will be removed (no -e given)", len(s.Envs)))
	} else {
		var lost []string
		for _, k := range sortedKeys(s.Envs) {
			if _, ok := p.Envs[k]; !ok {
				lost = append(lost, k)
			}
		}
		if len(lost) > 0 {
			w = append(w, "env variable(s) not given will be removed: "+strings.Join(lost, ", "))
		}
	}
	if (len(s.Ports) > 0 || len(s.URLs) > 0) && len(p.Ports) == 0 && len(p.HTTPPorts) == 0 {
		w = append(w, "the session's published ports will be removed (no --ports or --http-ports given)")
	}
	hasSSH := s.SSH.Direct != nil || s.SSH.Proxy != nil || s.SSH.Web != ""
	if hasSSH && p.SSH == nil {
		w = append(w, "SSH is on now; without --ssh it follows the app's setting (off for a plain image)")
	}
	if s.FM != "" && p.FM == nil {
		w = append(w, "the file manager is on now; without --fm it follows the app's setting (off for a plain image)")
	}
	return w
}

// restartLeaveReadyTimeout bounds how long a restart --wait waits for the
// session to leave the ready state before following it as usual.
const restartLeaveReadyTimeout = 15 * time.Second

// waitLeftReady polls a restarted session until it is no longer ready (or is
// gone), at most for timeout. Errors are left to the following wait.
func waitLeftReady(ctx context.Context, c *octaspace.Client, uuid string, timeout time.Duration) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	for {
		s, err := c.Sessions.Get(ctx, uuid)
		if err != nil || !s.IsReady() {
			return
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Second):
		}
	}
}

// waitReady polls the session until it is ready, printing each new status
// to stderr when verbose.
func waitReady(ctx context.Context, c *octaspace.Client, uuid string, timeout time.Duration, verbose bool) (*octaspace.Session, error) {
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	last := ""
	session, err := c.Sessions.WaitReady(ctx, uuid, &octaspace.WaitOptions{
		OnProgress: func(s *octaspace.Session) {
			if status := ui.SessionStatus(*s); verbose && status != last {
				fmt.Fprintf(os.Stderr, "  %s\n", status)
				last = status
			}
		},
	})
	switch {
	case errors.Is(err, octaspace.ErrSessionStopped):
		return nil, fmt.Errorf("session %s stopped before it became ready (its setup may have failed)", uuid)
	case errors.Is(err, context.DeadlineExceeded):
		return nil, fmt.Errorf("session %s is not ready after %s; it keeps starting, check 'octa sessions info %s'", uuid, timeout, uuid)
	case err != nil:
		return nil, client.Friendly(err)
	}
	return session, nil
}

// printSessionAccess prints how to reach a ready session.
func printSessionAccess(s octaspace.Session) {
	fmt.Println("Session is ready.")
	if a := ui.SSHAddress(s.SSH.Direct); a != "" {
		fmt.Printf("SSH:          ssh %s\n", a)
	}
	if a := ui.SSHAddress(s.SSH.Proxy); a != "" {
		fmt.Printf("SSH (proxy):  ssh %s\n", a)
	}
	if s.SSH.Web != "" {
		fmt.Printf("Web terminal: %s\n", s.SSH.Web)
	}
	if s.FM != "" {
		fmt.Printf("File manager: %s\n", s.FM)
	}
	ports := make([]string, 0, len(s.URLs))
	for port := range s.URLs {
		ports = append(ports, port)
	}
	sort.Strings(ports)
	for _, port := range ports {
		fmt.Printf("HTTP %-8s %s\n", port+":", s.URLs[port])
	}
}

// filterMachines returns the machines whose CPU or GPU model, country or
// city contains query (case-insensitive).
func filterMachines(machines []octaspace.ComputeMachine, query string) []octaspace.ComputeMachine {
	query = strings.ToLower(query)
	var filtered []octaspace.ComputeMachine
	for _, m := range machines {
		if strings.Contains(strings.ToLower(m.CPUModelName), query) ||
			strings.Contains(strings.ToLower(m.Country), query) ||
			strings.Contains(strings.ToLower(m.City), query) {
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
	return filtered
}

// sortByGPUCount orders machines by descending GPU count (most GPUs first).
func sortByGPUCount(machines []octaspace.ComputeMachine) {
	sort.SliceStable(machines, func(i, j int) bool {
		return len(machines[i].GPUs) > len(machines[j].GPUs)
	})
}

func init() {
	computeCmd.Flags().StringP("output", "o", "table", "Output format: table or json")
	computeSearchCmd.Flags().StringP("output", "o", "table", "Output format: table or json")

	computeStartCmd.Flags().StringP("output", "o", "table", "Output format: table or json")
	computeStartCmd.Flags().StringArray("node", nil, "Node number from 'octa compute', e.g. 7KQ4-M2XD-4; repeat for a batch (required)")
	computeStartCmd.Flags().StringP("from-file", "f", "", "JSON array of start requests for a batch (see 'octa compute start --help')")
	computeStartCmd.Flags().String("image", "", "Container image, e.g. ubuntu:24.04 (required)")
	computeStartCmd.Flags().String("app", "", "Application UUID; its settings decide --ssh and --fm when not given")
	addSessionParamFlags(computeStartCmd)
	addWaitFlags(computeStartCmd)

	computeRestartCmd.Flags().StringP("output", "o", "table", "Output format: table or json")
	computeRestartCmd.Flags().Bool("replace", false, "Confirm that the parameter flags replace all of the session's parameters")
	addSessionParamFlags(computeRestartCmd)
	addWaitFlags(computeRestartCmd)

	computeLogsCmd.Flags().StringP("output", "o", "table", "Output format: table or json")
	computeLogsCmd.Flags().Int("tail", 0, "Number of the latest container log lines, 1..1000 (API default: 200)")

	computeConnectCmd.Flags().Bool("proxy", false, "Force connection via proxy instead of direct SSH")

	computeCmd.AddCommand(computeSearchCmd)
	computeCmd.AddCommand(computeStartCmd)
	computeCmd.AddCommand(computeRestartCmd)
	computeCmd.AddCommand(computeLogsCmd)
	computeCmd.AddCommand(computeConnectCmd)
}
