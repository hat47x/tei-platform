package runtime_test

import (
	"context"
	"encoding/json"
	"os"
	"runtime"
	"strings"
	"testing"

	pruntime "tei/internal/plugin/runtime"
)

func TestStdioProcessLauncherRejectsAmbiguousHostEnvironmentBeforeSpawn(t *testing.T) {
	t.Setenv("TEI_PROCESS_HOST_SECRET_CANARY", "host-secret-do-not-disclose")
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	command := pruntime.ProcessCommand{Executable: executable, Args: []string{"--tei-process-env-probe"}}
	for _, tc := range []struct {
		name string
		env  []string
	}{
		{name: "duplicate PATH", env: []string{"PATH=/approved", "PATH=/unapproved"}},
		{name: "duplicate explicit secret", env: []string{"TEI_PROCESS_EXPLICIT_ALLOWLIST=first", "TEI_PROCESS_EXPLICIT_ALLOWLIST=second"}},
		{name: "missing assignment", env: []string{"TEI_PROCESS_EXPLICIT_ALLOWLIST"}},
		{name: "empty variable name", env: []string{"=hidden"}},
		{name: "blank variable name", env: []string{"  =hidden"}},
		{name: "newline variable name", env: []string{"BAD\nNAME=hidden"}},
		{name: "nul variable name", env: []string{"BAD\x00NAME=hidden"}},
		{name: "nul variable value", env: []string{"TEI_PROCESS_EXPLICIT_ALLOWLIST=secret\x00value"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			launcher := pruntime.StdioProcessLauncher{Env: tc.env}
			_, err := launcher.Invoke(context.Background(), command, processRequest())
			if err == nil || !strings.Contains(err.Error(), "environment") {
				t.Fatalf("invalid environment unexpectedly accepted: %v", err)
			}
			if strings.Contains(err.Error(), "hidden") ||
				strings.Contains(err.Error(), "unapproved") ||
				strings.Contains(err.Error(), "secret") ||
				strings.Contains(err.Error(), "host-secret-do-not-disclose") {
				t.Fatalf("environment diagnostic contains an untrusted value")
			}
		})
	}
}

func TestProcessEnvironmentKeysFollowOSIdentityRules(t *testing.T) {
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	launcher := pruntime.StdioProcessLauncher{
		Env: []string{"TEI_PROCESS_EXPLICIT_ALLOWLIST=first", "tei_process_explicit_allowlist=second"},
	}
	_, err = launcher.Invoke(context.Background(), pruntime.ProcessCommand{
		Executable: executable, Args: []string{"--tei-process-env-probe"},
	}, processRequest())
	if runtime.GOOS == "windows" {
		if err == nil || !strings.Contains(err.Error(), "duplicate") {
			t.Fatalf("Windows environment keys must be case-insensitive: %v", err)
		}
	} else if err != nil {
		t.Fatalf("POSIX allows case-distinct environment names: %v", err)
	}
}

func TestStdioProcessLauncherPreservesExplicitValueWithEquals(t *testing.T) {
	t.Setenv("TEI_PROCESS_HOST_SECRET_CANARY", "host-canary-not-forwarded")
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	launcher := pruntime.StdioProcessLauncher{Env: []string{"TEI_PROCESS_EXPLICIT_ALLOWLIST=approved=value"}}
	resp, err := launcher.Invoke(context.Background(), pruntime.ProcessCommand{
		Executable: executable, Args: []string{"--tei-process-env-probe"},
	}, processRequest())
	if err != nil {
		t.Fatal(err)
	}
	var probe struct {
		HostSecretPresent bool   `json:"hostSecretPresent"`
		Explicit          string `json:"explicit"`
	}
	if err := json.Unmarshal(resp.Payload, &probe); err != nil {
		t.Fatal(err)
	}
	if probe.HostSecretPresent || probe.Explicit != "approved=value" {
		t.Fatalf("host environment was leaked or explicit value was rewritten: %+v", probe)
	}
}
