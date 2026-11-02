package cli

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func TestParseComputeBatch(t *testing.T) {
	items, err := parseComputeBatch([]byte(`[
		{"id": 7, "node_number": "7KQ4-M2XD-4", "image": "ubuntu:24.04", "ssh": false, "envs": {"A": "1"}},
		{"id": "web", "node_number": "M95S-WZG8-7", "image": "nginx", "http_ports": [80]},
		{"node_number": "WJFM-MYWR-9", "image": "alpine"}
	]`))
	if err != nil {
		t.Fatal(err)
	}
	if got := []string{items[0].ID, items[1].ID, items[2].ID}; strings.Join(got, ",") != "7,web,3" {
		t.Errorf("ids = %v", got)
	}
	if items[0].SSH == nil || *items[0].SSH || items[0].Envs["A"] != "1" {
		t.Errorf("explicit ssh=false or envs lost: %+v", items[0].ComputeStartParams)
	}
	if items[1].HTTPPorts[0] != 80 {
		t.Errorf("http_ports lost: %+v", items[1].ComputeStartParams)
	}
}

func TestParseComputeBatchErrors(t *testing.T) {
	cases := map[string]string{
		`{}`: "JSON array",
		`[]`: "empty",
		`[{"node_number":"A","image":"x","disk":5}]`: "unknown field",
		`[{"node_number":"A"}]`:                      "image is required",
		`[{"id":1,"node_number":"A","image":"x"},{"id":"1","node_number":"B","image":"x"}]`: "duplicate id",
		`[{"id":true,"node_number":"A","image":"x"}]`:                                       "id must be",
		`[{"node_number":"A","image":"x","http_ports":[7682]}]`:                             "reserved",
	}
	for in, want := range cases {
		if _, err := parseComputeBatch([]byte(in)); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("parseComputeBatch(%s) = %v, want %q", in, err, want)
		}
	}
}

// fakeAPI serves POST /services/compute and /services/render and records the
// last request body.
type fakeAPI struct {
	body []byte
	auth string
}

func (f *fakeAPI) start(t *testing.T) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.body, _ = io.ReadAll(r.Body)
		f.auth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		if bytes.HasPrefix(bytes.TrimSpace(f.body), []byte("{")) {
			w.WriteHeader(http.StatusCreated)
			io.WriteString(w, `{"uuid":"11111111-1111-1111-1111-111111111111"}`)
			return
		}
		var items []map[string]any
		json.Unmarshal(f.body, &items)
		var out []map[string]any
		for i, it := range items {
			if i == len(items)-1 && len(items) > 1 {
				out = append(out, map[string]any{"id": it["id"], "status": 1, "reason": "Node not available"})
				continue
			}
			out = append(out, map[string]any{"id": it["id"], "status": 0, "uuid": "22222222-2222-2222-2222-22222222222" + string(rune('0'+i))})
		}
		json.NewEncoder(w).Encode(out)
	}))
	t.Cleanup(srv.Close)

	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("OCTA_API_URL", srv.URL)
	os.MkdirAll(filepath.Join(dir, "octa"), 0o700)
	os.WriteFile(filepath.Join(dir, "octa", "next.yaml"), []byte("api_key: octa_ak_test\n"), 0o600)
}

func runCLI(t *testing.T, args ...string) error {
	t.Helper()
	// Reset flags that previous runs may have set on the shared commands.
	for _, c := range []*cobraCmd{computeStartCmd, computeRestartCmd, renderStartCmd} {
		c.Flags().VisitAll(func(f *flagT) {
			if f.Changed {
				f.Value.Set(f.DefValue)
				if sv, ok := f.Value.(interface{ Replace([]string) error }); ok {
					sv.Replace(nil)
				}
				f.Changed = false
			}
		})
	}
	rootCmd.SetArgs(args)
	rootCmd.SetOut(io.Discard)
	return rootCmd.Execute()
}

func TestComputeStartSingleSendsObject(t *testing.T) {
	var api fakeAPI
	api.start(t)
	if err := runCLI(t, "compute", "start", "--node", "7KQ4-M2XD-4", "--image", "alpine", "-o", "json"); err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(api.body, []byte("{")) || api.auth != "Bearer octa_ak_test" {
		t.Errorf("single start body = %s, auth = %q", api.body, api.auth)
	}
}

func TestComputeStartBatchSendsArray(t *testing.T) {
	var api fakeAPI
	api.start(t)
	err := runCLI(t, "compute", "start", "--node", "7KQ4-M2XD-4", "--node", "M95S-WZG8-7", "--image", "alpine", "-e", "A=1", "-o", "json")
	if err == nil || !strings.Contains(err.Error(), "1 of 2 requests failed") {
		t.Fatalf("err = %v", err)
	}
	var items []map[string]any
	if err := json.Unmarshal(api.body, &items); err != nil || len(items) != 2 {
		t.Fatalf("batch body = %s", api.body)
	}
	if items[0]["id"] != "1" || items[1]["node_number"] != "M95S-WZG8-7" || items[1]["image"] != "alpine" {
		t.Errorf("batch items = %v", items)
	}
}

func TestRenderStartBatch(t *testing.T) {
	var api fakeAPI
	api.start(t)
	if err := runCLI(t, "render", "start", "--node", "A", "--node", "B", "--node", "C", "-o", "json"); err == nil {
		t.Fatal("a failed request must fail the command")
	}
	var items []map[string]any
	json.Unmarshal(api.body, &items)
	if len(items) != 3 || items[2]["node_number"] != "C" || len(items[0]) != 2 {
		t.Errorf("render batch = %s", api.body)
	}
}

func TestStartRejectsStrayArgs(t *testing.T) {
	var api fakeAPI
	api.start(t)
	err := runCLI(t, "compute", "start", "--node", "wjfm", "mywr", "9", "--image", "alpine")
	if err == nil || !strings.Contains(err.Error(), "quote flag values") || api.body != nil {
		t.Errorf("err = %v, request sent = %s", err, api.body)
	}
}

func TestRestartRequiresReplace(t *testing.T) {
	var api fakeAPI
	api.start(t)
	err := runCLI(t, "compute", "restart", "7c6e", "--args", "x")
	if err == nil || !strings.Contains(err.Error(), "--replace") || api.body != nil {
		t.Errorf("err = %v, request sent = %s", err, api.body)
	}
	err = runCLI(t, "compute", "restart", "7c6e", "--args", "x", "--replace")
	if err == nil || !strings.Contains(err.Error(), "--disk is required") || api.body != nil {
		t.Errorf("--replace without --disk: err = %v, request sent = %s", err, api.body)
	}
	err = runCLI(t, "compute", "restart", "7c6e", "--replace")
	if err == nil || !strings.Contains(err.Error(), "needs the parameter flags") {
		t.Errorf("--replace alone: err = %v", err)
	}
}

func TestStartFromFileConflicts(t *testing.T) {
	err := runCLI(t, "compute", "start", "--from-file", "x.json", "--image", "alpine")
	if err == nil || !strings.Contains(err.Error(), "cannot be combined with --image") {
		t.Errorf("err = %v", err)
	}
}

type (
	cobraCmd = cobra.Command
	flagT    = pflag.Flag
)

func TestListingCommandsRejectTypos(t *testing.T) {
	err := runCLI(t, "compute", "serach", "3090")
	if err == nil || !strings.Contains(err.Error(), `unknown command "serach"`) || !strings.Contains(err.Error(), "search") {
		t.Errorf("err = %v", err)
	}
}
