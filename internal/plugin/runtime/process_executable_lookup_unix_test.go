//go:build !windows

package runtime_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	pruntime "tei/internal/plugin/runtime"
)

// os/exec searches the parent PATH before Cmd.Env is applied. This regression
// proves an unqualified plugin command never executes a Host PATH candidate
// unless that directory is approved in the child's explicit environment.
func TestStdioProcessLauncherExecutableLookupUsesExplicitPathOnly(t *testing.T) {
	hostDir := t.TempDir()
	approvedDir := t.TempDir()
	const name = "tei-lookup-probe"
	hostExecutable := filepath.Join(hostDir, name)
	// If the host PATH is consulted, this script runs instead of the trusted
	// test binary and the plugin handshake necessarily fails.
	if err := os.WriteFile(hostExecutable, []byte("#!/bin/sh\nexit 61\n"), 0700); err != nil {
		t.Fatal(err)
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(executable, filepath.Join(approvedDir, name)); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", hostDir)
	command := pruntime.ProcessCommand{Executable: name, Args: []string{"--tei-process-env-probe"}}
	for _, tt := range []struct {
		name string
		env  []string
	}{
		{name: "nil environment"},
		{name: "empty environment", env: []string{}},
		{name: "unapproved relative PATH", env: []string{"PATH=."}},
		{name: "explicit PATH without executable", env: []string{"PATH=" + filepath.Join(approvedDir, "missing")}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			launcher := pruntime.StdioProcessLauncher{Env: tt.env}
			if _, err := launcher.Invoke(context.Background(), command, processRequest()); err == nil {
				t.Fatal("unqualified executable unexpectedly resolved without an approved PATH candidate")
			} else if !strings.Contains(err.Error(), "explicit") {
				t.Fatalf("unqualified executable was not fail-closed: %v", err)
			}
		})
	}
	// On Unix, a backslash is a regular filename byte, not a path
	// separator. os/exec would still search the parent PATH for this name.
	backslashName := "tei\\\\host-only-probe"
	if err := os.WriteFile(filepath.Join(hostDir, backslashName), []byte("#!/bin/sh\\nexit 61\\n"), 0700); err != nil {
		t.Fatal(err)
	}
	if _, err := (pruntime.StdioProcessLauncher{}).Invoke(context.Background(),
		pruntime.ProcessCommand{Executable: backslashName}, processRequest()); err == nil || !strings.Contains(err.Error(), "explicit") {
		t.Fatal("Unix backslash command name bypassed explicit PATH admission")
	}
	launcher := pruntime.StdioProcessLauncher{Env: []string{
		"PATH=" + approvedDir,
		"TEI_PROCESS_EXPLICIT_ALLOWLIST=approved",
	}}
	resp, err := launcher.Invoke(context.Background(), command, processRequest())
	if err != nil {
		t.Fatalf("host-approved PATH command did not run: %v", err)
	}
	checkProcessEnvProbe(t, resp, "approved")

	// An explicit absolute command must work even with an empty environment.
	explicit := pruntime.StdioProcessLauncher{}
	resp, err = explicit.Invoke(context.Background(), pruntime.ProcessCommand{
		Executable: executable, Args: []string{"--tei-process-env-probe"},
	}, processRequest())
	if err != nil {
		t.Fatalf("absolute executable was rejected: %v", err)
	}
	checkProcessEnvProbe(t, resp, "")
}
