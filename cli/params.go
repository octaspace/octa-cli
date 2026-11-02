package cli

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"

	octaspace "github.com/octaspace/go-sdk"
	"github.com/spf13/cobra"
)

// API limits of the compute session parameters, checked before a request so
// the user gets every problem at once instead of one 400 at a time. The
// server stays the source of truth.
const (
	maxCommandLineBytes   = 4096      // entrypoint, args
	maxStartupScriptBytes = 64 * 1024 // startup_script
	maxEnvs               = 64
	maxEnvValueBytes      = 4096
)

// reservedHTTPPorts are taken by the web terminal and the file manager.
var reservedHTTPPorts = map[int]bool{7681: true, 7682: true}

var envKeyRe = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// sessionParamsInput holds the values of the session parameter flags shared
// by 'compute start' and 'compute restart'. The *Set fields record which
// flags were given, since the API distinguishes "not set" from a zero value
// for SSH and FM.
type sessionParamsInput struct {
	DiskSize      int
	Envs          []string // KEY=VALUE
	Ports         []int
	HTTPPorts     []int
	Entrypoint    string
	Args          string
	StartupScript string // the script, or @path to read it from
	SSH           bool
	SSHSet        bool
	FM            bool
	FMSet         bool
}

// sessionParams are the parsed and validated session parameters.
type sessionParams struct {
	DiskSize      int
	Envs          map[string]string
	Ports         []int
	HTTPPorts     []int
	Entrypoint    string
	Args          string
	StartupScript string
	SSH           *bool
	FM            *bool
}

// empty reports whether no parameter is set.
func (p *sessionParams) empty() bool {
	return p.DiskSize == 0 && len(p.Envs) == 0 && len(p.Ports) == 0 &&
		len(p.HTTPPorts) == 0 && p.Entrypoint == "" && p.Args == "" &&
		p.StartupScript == "" && p.SSH == nil && p.FM == nil
}

func (p *sessionParams) startParams(nodeNumber, image, app string) *octaspace.ComputeStartParams {
	return &octaspace.ComputeStartParams{
		NodeNumber: nodeNumber, Image: image, App: app,
		DiskSize: p.DiskSize, Envs: p.Envs, Ports: p.Ports, HTTPPorts: p.HTTPPorts,
		Entrypoint: p.Entrypoint, Args: p.Args, StartupScript: p.StartupScript,
		SSH: p.SSH, FM: p.FM,
	}
}

func (p *sessionParams) restartParams() *octaspace.ComputeRestartParams {
	if p.empty() {
		return nil
	}
	return &octaspace.ComputeRestartParams{
		DiskSize: p.DiskSize, Envs: p.Envs, Ports: p.Ports, HTTPPorts: p.HTTPPorts,
		Entrypoint: p.Entrypoint, Args: p.Args, StartupScript: p.StartupScript,
		SSH: p.SSH, FM: p.FM,
	}
}

// sessionParamFlags are the flags registered by addSessionParamFlags.
var sessionParamFlags = []string{"disk", "env", "ports", "http-ports", "entrypoint", "args", "startup-script", "ssh", "fm"}

// addSessionParamFlags registers the session parameter flags on cmd.
func addSessionParamFlags(cmd *cobra.Command) {
	f := cmd.Flags()
	f.Int("disk", 0, "Disk size in GB (API default: 1)")
	f.StringArrayP("env", "e", nil, "Environment variable KEY=VALUE (repeatable)")
	f.IntSlice("ports", nil, "TCP/UDP ports to expose, comma-separated (22 is always exposed)")
	f.IntSlice("http-ports", nil, "HTTP ports to expose, comma-separated (7681 and 7682 are reserved)")
	f.String("entrypoint", "", "Replace the image ENTRYPOINT (one line, split like a shell command, nothing expanded)")
	f.String("args", "", "Replace the image CMD (one line)")
	f.String("startup-script", "", "Script run as root after start; @path reads it from a file")
	f.Bool("ssh", false, "Set up SSH and a web terminal (--ssh=false to turn off; default: the app's setting)")
	f.Bool("fm", false, "Start a web file manager without a login (--fm=false to turn off; default: the app's setting)")
}

// sessionParamsFromFlags reads the flags registered by addSessionParamFlags.
func sessionParamsFromFlags(cmd *cobra.Command) sessionParamsInput {
	f := cmd.Flags()
	var in sessionParamsInput
	in.DiskSize, _ = f.GetInt("disk")
	in.Envs, _ = f.GetStringArray("env")
	in.Ports, _ = f.GetIntSlice("ports")
	in.HTTPPorts, _ = f.GetIntSlice("http-ports")
	in.Entrypoint, _ = f.GetString("entrypoint")
	in.Args, _ = f.GetString("args")
	in.StartupScript, _ = f.GetString("startup-script")
	in.SSH, _ = f.GetBool("ssh")
	in.SSHSet = f.Changed("ssh")
	in.FM, _ = f.GetBool("fm")
	in.FMSet = f.Changed("fm")
	return in
}

// buildSessionParams parses and validates the session parameter flags.
func buildSessionParams(in sessionParamsInput) (*sessionParams, error) {
	p := &sessionParams{
		DiskSize: in.DiskSize, Ports: in.Ports, HTTPPorts: in.HTTPPorts,
		Entrypoint: in.Entrypoint, Args: in.Args,
	}
	if in.SSHSet {
		p.SSH = octaspace.Bool(in.SSH)
	}
	if in.FMSet {
		p.FM = octaspace.Bool(in.FM)
	}

	var errs []error
	if in.DiskSize < 0 {
		errs = append(errs, errors.New("--disk must be at least 1 GB"))
	}

	envs, err := parseEnvs(in.Envs)
	if err != nil {
		errs = append(errs, err)
	}
	p.Envs = envs

	script, err := readStartupScript(in.StartupScript)
	if err != nil {
		errs = append(errs, err)
	}
	p.StartupScript = script

	errs = append(errs, checkCommandLine("--entrypoint", p.Entrypoint)...)
	errs = append(errs, checkCommandLine("--args", p.Args)...)
	if len(p.StartupScript) > maxStartupScriptBytes {
		errs = append(errs, fmt.Errorf("--startup-script is longer than %d KiB", maxStartupScriptBytes/1024))
	}
	errs = append(errs, checkPorts(p.Ports, p.HTTPPorts)...)

	if err := errors.Join(errs...); err != nil {
		return nil, err
	}
	return p, nil
}

// parseEnvs parses KEY=VALUE pairs; a later key overrides an earlier one.
func parseEnvs(pairs []string) (map[string]string, error) {
	if len(pairs) == 0 {
		return nil, nil
	}
	envs := make(map[string]string, len(pairs))
	var errs []error
	for _, pair := range pairs {
		key, value, ok := strings.Cut(pair, "=")
		if !ok {
			errs = append(errs, fmt.Errorf("invalid env %q, expected KEY=VALUE", pair))
			continue
		}
		envs[key] = value
	}
	errs = append(errs, checkEnvs(envs)...)
	return envs, errors.Join(errs...)
}

// checkEnvs checks env names, value sizes and the number of variables.
func checkEnvs(envs map[string]string) []error {
	var errs []error
	for _, key := range sortedKeys(envs) {
		switch {
		case !envKeyRe.MatchString(key):
			errs = append(errs, fmt.Errorf("invalid env name %q: use letters, digits and _, not starting with a digit", key))
		case len(envs[key]) > maxEnvValueBytes:
			errs = append(errs, fmt.Errorf("env %s is longer than %d bytes", key, maxEnvValueBytes))
		}
	}
	if len(envs) > maxEnvs {
		errs = append(errs, fmt.Errorf("at most %d env variables are allowed, got %d", maxEnvs, len(envs)))
	}
	return errs
}

// checkStartParams checks the API limits of one start request.
func checkStartParams(p *octaspace.ComputeStartParams) []error {
	var errs []error
	if strings.TrimSpace(p.NodeNumber) == "" {
		errs = append(errs, errors.New("node_number is required"))
	}
	if strings.TrimSpace(p.Image) == "" {
		errs = append(errs, errors.New("image is required"))
	}
	if p.DiskSize < 0 {
		errs = append(errs, errors.New("disk_size must be at least 1 GB"))
	}
	errs = append(errs, checkEnvs(p.Envs)...)
	errs = append(errs, checkCommandLine("entrypoint", p.Entrypoint)...)
	errs = append(errs, checkCommandLine("args", p.Args)...)
	if len(p.StartupScript) > maxStartupScriptBytes {
		errs = append(errs, fmt.Errorf("startup_script is longer than %d KiB", maxStartupScriptBytes/1024))
	}
	errs = append(errs, checkPorts(p.Ports, p.HTTPPorts)...)
	return errs
}

// readStartupScript returns the script, reading it from a file for @path.
func readStartupScript(value string) (string, error) {
	path, ok := strings.CutPrefix(value, "@")
	if !ok {
		return value, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("--startup-script: %w", err)
	}
	return string(data), nil
}

func checkCommandLine(flag, value string) []error {
	var errs []error
	if len(value) > maxCommandLineBytes {
		errs = append(errs, fmt.Errorf("%s is longer than %d bytes", flag, maxCommandLineBytes))
	}
	if strings.ContainsAny(value, "\n\r") {
		errs = append(errs, fmt.Errorf("%s must be a single line", flag))
	}
	return errs
}

func checkPorts(ports, httpPorts []int) []error {
	var errs []error
	seen := map[int]bool{}
	for _, port := range ports {
		if port < 1 || port > 65535 {
			errs = append(errs, fmt.Errorf("--ports: %d is not a port (1..65535)", port))
		}
		seen[port] = true
	}
	for _, port := range httpPorts {
		switch {
		case port < 1 || port > 65535:
			errs = append(errs, fmt.Errorf("--http-ports: %d is not a port (1..65535)", port))
		case reservedHTTPPorts[port]:
			errs = append(errs, fmt.Errorf("--http-ports: %d is reserved for the web terminal and file manager", port))
		case seen[port]:
			errs = append(errs, fmt.Errorf("port %d is in both --ports and --http-ports", port))
		}
	}
	return errs
}
