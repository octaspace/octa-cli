package cli

import (
	"reflect"
	"testing"

	octaspace "github.com/octaspace/go-sdk"
)

func TestBuildDeployParams_InheritsTemplateAndOverlaysUserEnvs(t *testing.T) {
	template := &octaspace.App{
		UUID: "app-uuid", Image: "template:latest", StartCommand: "run-template",
		Envs:  map[string]any{"STRING": "default", "BOOL": false, "NUMBER": float64(3), "EMPTY": nil},
		Ports: []int{22}, HTTPPorts: []int{8080},
		Extra: map[string]any{"min_disk_size": float64(25)},
	}
	params, err := buildDeployParams(deployParamsInput{
		NodeID: 7, AppUUID: template.UUID, Template: template,
		Envs: "STRING=override,USER=value", EnvsSet: true,
	})
	if err != nil {
		t.Fatalf("buildDeployParams: %v", err)
	}
	if params.Image != "template:latest" || params.DiskSize != 25 || params.StartCommand != "run-template" {
		t.Errorf("template scalar defaults lost: %+v", params)
	}
	if !reflect.DeepEqual(params.Ports, []int{22}) || !reflect.DeepEqual(params.HTTPPorts, []int{8080}) {
		t.Errorf("template port defaults lost: %+v", params)
	}
	wantEnvs := map[string]string{"STRING": "override", "BOOL": "false", "NUMBER": "3", "EMPTY": "", "USER": "value"}
	if !reflect.DeepEqual(params.Envs, wantEnvs) {
		t.Errorf("Envs = %#v, want %#v", params.Envs, wantEnvs)
	}
}

func TestBuildDeployParams_ExplicitFlagsOverrideTemplate(t *testing.T) {
	template := &octaspace.App{
		Image: "template", StartCommand: "template-command",
		Ports: []int{22}, HTTPPorts: []int{8080},
		Extra: map[string]any{"min_disk_size": float64(25)},
	}
	params, err := buildDeployParams(deployParamsInput{
		Template: template,
		Image:    "custom", ImageSet: true,
		DiskSize: 50, DiskSet: true,
		Ports: []int{1000}, PortsSet: true,
		HTTPPorts: []int{}, HTTPPortsSet: true,
		StartCommand: "", StartCommandSet: true,
		Entrypoint: "/bin/sh",
	})
	if err != nil {
		t.Fatalf("buildDeployParams: %v", err)
	}
	if params.Image != "custom" || params.DiskSize != 50 || params.StartCommand != "" || params.Entrypoint != "/bin/sh" {
		t.Errorf("explicit scalar flags lost: %+v", params)
	}
	if !reflect.DeepEqual(params.Ports, []int{1000}) || len(params.HTTPPorts) != 0 {
		t.Errorf("explicit port flags lost: %+v", params)
	}
}

func TestBuildDeployParams_RejectsMalformedEnv(t *testing.T) {
	_, err := buildDeployParams(deployParamsInput{Envs: "INVALID", EnvsSet: true})
	if err == nil {
		t.Fatal("malformed env error = nil")
	}
}
