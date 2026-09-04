package main

import (
	"errors"
	"strings"
	"testing"
)

func TestInspectGoEnvironmentAcceptsModuleOnlyExecution(t *testing.T) {
	tests := []struct {
		name   string
		goWork string
	}{
		{name: "workspace未使用", goWork: ""},
		{name: "GOWORK off", goWork: "off\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			environment, err := inspectGoEnvironment(fakeGoEnvironmentOutput(tt.goWork, ""))
			if err != nil {
				t.Fatalf("inspectGoEnvironment() error=%v", err)
			}
			if environment.ExternalWorkspaceActive {
				t.Fatalf("externalWorkspaceActive=true, want false: %#v", environment)
			}
			if !environment.GoFlagsEmpty {
				t.Fatalf("goFlagsEmpty=false, want true: %#v", environment)
			}
		})
	}
}

func TestInspectGoEnvironmentRejectsExternalWorkspaceWithoutPersistingPath(t *testing.T) {
	const workspacePath = "/private/build/go.work"
	environment, err := inspectGoEnvironment(fakeGoEnvironmentOutput(workspacePath+"\n", ""))
	if err == nil {
		t.Fatal("inspectGoEnvironment() error=nil, want external workspace failure")
	}
	if !environment.ExternalWorkspaceActive {
		t.Fatalf("externalWorkspaceActive=false, want true: %#v", environment)
	}
	if strings.Contains(err.Error(), workspacePath) {
		t.Fatalf("external workspace pathをerrorへ露出しています: %q", err)
	}
}

func TestInspectGoEnvironmentRejectsGoFlagsWithoutPersistingValue(t *testing.T) {
	const goFlags = "-overlay=/private/build/overlay.json -tags=evidence_override"
	environment, err := inspectGoEnvironment(fakeGoEnvironmentOutput("", goFlags+"\n"))
	if err == nil {
		t.Fatal("inspectGoEnvironment() error=nil, want GOFLAGS failure")
	}
	if environment.ExternalWorkspaceActive {
		t.Fatalf("externalWorkspaceActive=true, want false: %#v", environment)
	}
	if environment.GoFlagsEmpty {
		t.Fatalf("goFlagsEmpty=true, want false: %#v", environment)
	}
	if strings.Contains(err.Error(), goFlags) || strings.Contains(err.Error(), "/private/build/overlay.json") {
		t.Fatalf("GOFLAGS本文をerrorへ露出しています: %q", err)
	}
}

func TestInspectGoEnvironmentFailsClosedWhenGoEnvCannotBeObserved(t *testing.T) {
	calls := 0
	_, err := inspectGoEnvironment(func(name string, args ...string) (string, error) {
		calls++
		if calls == 1 {
			return "", nil
		}
		return "", errors.New("go env failed")
	})
	if err == nil {
		t.Fatal("GOFLAGS観測失敗をerrorにしていません")
	}
}

func fakeGoEnvironmentOutput(goWork, goFlags string) commandOutputFunc {
	return func(name string, args ...string) (string, error) {
		if name != "go" || len(args) != 2 || args[0] != "env" {
			return "", errors.New("unexpected command")
		}
		switch args[1] {
		case "GOWORK":
			return goWork, nil
		case "GOFLAGS":
			return goFlags, nil
		default:
			return "", errors.New("unexpected go env key")
		}
	}
}
