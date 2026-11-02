package cli

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	octaspace "github.com/octaspace/go-sdk"
)

func TestBuildSessionParams_TriStateFlags(t *testing.T) {
	p, err := buildSessionParams(sessionParamsInput{})
	if err != nil {
		t.Fatal(err)
	}
	if p.SSH != nil || p.FM != nil || !p.empty() {
		t.Errorf("unset flags must stay unset: %+v", p)
	}
	if p.restartParams() != nil {
		t.Error("empty restart params must be nil (restart with the original parameters)")
	}

	p, err = buildSessionParams(sessionParamsInput{SSH: false, SSHSet: true, FM: true, FMSet: true})
	if err != nil {
		t.Fatal(err)
	}
	if p.SSH == nil || *p.SSH || p.FM == nil || !*p.FM {
		t.Errorf("explicit --ssh=false --fm lost: ssh=%v fm=%v", p.SSH, p.FM)
	}
	if p.restartParams() == nil {
		t.Error("restart params with --ssh=false must be sent")
	}
}

func TestBuildSessionParams_StartParams(t *testing.T) {
	p, err := buildSessionParams(sessionParamsInput{
		DiskSize: 20, Envs: []string{"A=1", "B=x=y", "A=2"},
		Ports: []int{8080}, HTTPPorts: []int{8888},
		Entrypoint: "sh -c", Args: "'sleep infinity'",
	})
	if err != nil {
		t.Fatal(err)
	}
	got := p.startParams("7KQ4-M2XD-4", "ubuntu:24.04", "")
	want := &octaspace.ComputeStartParams{
		NodeNumber: "7KQ4-M2XD-4", Image: "ubuntu:24.04", DiskSize: 20,
		Envs:  map[string]string{"A": "2", "B": "x=y"},
		Ports: []int{8080}, HTTPPorts: []int{8888},
		Entrypoint: "sh -c", Args: "'sleep infinity'",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("start params = %+v, want %+v", got, want)
	}
}

func TestBuildSessionParams_Validation(t *testing.T) {
	_, err := buildSessionParams(sessionParamsInput{
		Envs:       []string{"NOEQUALS", "1BAD=x"},
		HTTPPorts:  []int{7681, 8080, 70000},
		Ports:      []int{8080},
		Entrypoint: "a\nb",
		Args:       strings.Repeat("x", maxCommandLineBytes+1),
	})
	if err == nil {
		t.Fatal("expected validation errors")
	}
	for _, part := range []string{"NOEQUALS", "1BAD", "7681", "both --ports and --http-ports", "70000", "--entrypoint must be a single line", "--args is longer"} {
		if !strings.Contains(err.Error(), part) {
			t.Errorf("error does not mention %q:\n%v", part, err)
		}
	}
}

func TestBuildSessionParams_TooManyEnvs(t *testing.T) {
	envs := make([]string, maxEnvs+1)
	for i := range envs {
		envs[i] = "K" + strings.Repeat("X", i) + "=v"
	}
	if _, err := buildSessionParams(sessionParamsInput{Envs: envs}); err == nil || !strings.Contains(err.Error(), "at most 64") {
		t.Errorf("err = %v", err)
	}
}

func TestBuildSessionParams_StartupScriptFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "onstart.sh")
	if err := os.WriteFile(path, []byte("#!/bin/sh\necho hi\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	p, err := buildSessionParams(sessionParamsInput{StartupScript: "@" + path})
	if err != nil {
		t.Fatal(err)
	}
	if p.StartupScript != "#!/bin/sh\necho hi\n" {
		t.Errorf("script = %q", p.StartupScript)
	}
	if _, err := buildSessionParams(sessionParamsInput{StartupScript: "@" + path + ".missing"}); err == nil {
		t.Error("missing script file must fail")
	}
}

func TestFilterMachines(t *testing.T) {
	machines := []octaspace.ComputeMachine{
		{NodeNumber: "A", CPUModelName: "AMD EPYC", City: "Kyiv"},
		{NodeNumber: "B", GPUs: []octaspace.MachineGPU{{Model: "NVIDIA GeForce RTX 4090"}}},
		{NodeNumber: "C", Country: "Germany"},
	}
	for query, want := range map[string]string{"epyc": "A", "4090": "B", "germ": "C", "kyiv": "A"} {
		got := filterMachines(machines, query)
		if len(got) != 1 || got[0].NodeNumber != want {
			t.Errorf("filter %q = %+v, want %s", query, got, want)
		}
	}
}

func TestSSHArgs(t *testing.T) {
	got, err := sshArgs(&octaspace.SSHEndpoint{Host: "px-uk-1.octa.computer", Port: 30103, User: "ubuntu"})
	want := []string{"ssh", "-p", "30103", "-l", "ubuntu", "--", "px-uk-1.octa.computer"}
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Errorf("sshArgs = %q, %v", got, err)
	}
	if got, _ := sshArgs(&octaspace.SSHEndpoint{Host: "203.0.113.10", Port: 22}); got[4] != "root" {
		t.Errorf("default user: %q", got)
	}
	for _, e := range []octaspace.SSHEndpoint{
		{Host: "h", Port: 22, User: "-oProxyCommand=touch /tmp/pwned"},
		{Host: "h", Port: 22, User: "a b"},
		{Host: "-oProxyCommand=x", Port: 22},
		{Host: "user@host", Port: 22},
		{Host: "h", Port: 70000},
	} {
		if _, err := sshArgs(&e); err == nil {
			t.Errorf("sshArgs(%+v) must be refused", e)
		}
	}
}

func TestRestartDropWarnings(t *testing.T) {
	s := octaspace.Session{
		Envs:  map[string]string{"A": "1", "B": "2"},
		Ports: map[string]int{"6006": 31006},
		SSH:   octaspace.SessionSSH{Proxy: &octaspace.SSHEndpoint{Host: "h", Port: 1}},
		FM:    "https://fm",
	}
	w := strings.Join(restartDropWarnings(s, &octaspace.ComputeRestartParams{DiskSize: 5, Envs: map[string]string{"A": "1"}}), "\n")
	for _, part := range []string{"removed: B", "published ports", "SSH is on", "file manager is on"} {
		if !strings.Contains(w, part) {
			t.Errorf("warnings do not mention %q:\n%s", part, w)
		}
	}
	full := &octaspace.ComputeRestartParams{
		DiskSize: 5, Envs: map[string]string{"A": "1", "B": "2"}, Ports: []int{6006},
		SSH: octaspace.Bool(true), FM: octaspace.Bool(false),
	}
	if w := restartDropWarnings(s, full); len(w) != 0 {
		t.Errorf("full set must not warn: %q", w)
	}
}
