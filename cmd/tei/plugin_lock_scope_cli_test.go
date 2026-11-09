package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"tei/internal/plugin"
	pluginruntime "tei/internal/plugin/runtime"
)

func TestPluginLockExplicitEnvironmentReferenceCLI(t *testing.T) {
	repo := testRepositoryRoot(t)
	bin := filepath.Join(t.TempDir(), "tei")
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}
	build := exec.Command("go", "build", "-o", bin, "./cmd/tei")
	build.Dir = repo
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build CLI: %v\n%s", err, out)
	}
	workspace := filepath.Join(repo, "reference", "workspace")
	run := func(exit int, args ...string) cliResultEnvelope {
		t.Helper()
		cmd := exec.Command(bin, args...)
		var out, stderr bytes.Buffer
		cmd.Stdout, cmd.Stderr = &out, &stderr
		err := cmd.Run()
		code := 0
		if err != nil {
			if e, ok := err.(*exec.ExitError); ok {
				code = e.ExitCode()
			} else {
				t.Fatal(err)
			}
		}
		if code != exit {
			t.Fatalf("%v exit=%d want=%d stdout=%s stderr=%s", args, code, exit, out.String(), stderr.String())
		}
		var envelope cliResultEnvelope
		if err := json.Unmarshal(out.Bytes(), &envelope); err != nil {
			t.Fatalf("invalid result: %v: %s", err, out.String())
		}
		return envelope
	}
	lock := func(env string, scoped bool) (cliResultEnvelope, plugin.Lock) {
		t.Helper()
		args := []string{"plugin-lock", "-workspace", workspace}
		if scoped {
			args = append(args, "-environment", env)
		}
		report := run(0, args...)
		var result plugin.Lock
		if err := reMarshal(report.Result, &result); err != nil {
			t.Fatal(err)
		}
		return report, result
	}
	allReport, all := lock("", false)
	partialReport, partial := lock("development", true)
	if !allReport.OK || allReport.Mode != "" || len(all.Environments) < 2 {
		t.Fatalf("default lock: %+v %+v", allReport, all)
	}
	if !partialReport.OK || partialReport.Mode != "environment" || len(partial.Environments) != 1 || string(partial.Environments[0].Environment) != "development" {
		t.Fatalf("partial lock: %+v %+v", partialReport, partial)
	}
	var expected plugin.EnvironmentLock
	for _, item := range all.Environments {
		if item.Environment == "development" {
			expected = item
		}
	}
	if !reflect.DeepEqual(expected, partial.Environments[0]) {
		t.Fatalf("selected environment differs from full lock: %+v %+v", expected, partial.Environments[0])
	}
	raw, err := json.Marshal(partial)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "development-lock.json")
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	fullRaw, err := json.Marshal(all)
	if err != nil {
		t.Fatal(err)
	}
	fullPath := filepath.Join(t.TempDir(), "all-lock.json")
	if err := os.WriteFile(fullPath, fullRaw, 0o644); err != nil {
		t.Fatal(err)
	}
	if got := run(0, "plugin-resolve", "-workspace", workspace, "-environment", "development", "-plugin-lock", path); !got.OK {
		t.Fatalf("selected resolve: %+v", got)
	}
	got := run(0, "plugin-runtime-plan", "-workspace", workspace, "-environment", "development", "-plugin-lock", path)
	if !got.OK {
		t.Fatalf("selected runtime plan: %+v", got)
	}
	fullPlan := run(0, "plugin-runtime-plan", "-workspace", workspace, "-environment", "development", "-plugin-lock", fullPath)
	var selectedPlan, wholePlan pluginruntime.EnvironmentPlan
	if err := reMarshal(got.Result, &selectedPlan); err != nil {
		t.Fatal(err)
	}
	if err := reMarshal(fullPlan.Result, &wholePlan); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(selectedPlan, wholePlan) {
		t.Fatalf("selected and full-lock runtime plans differ: %+v / %+v", selectedPlan, wholePlan)
	}
	// A lock that does not describe the requested environment is the caller's
	// input problem, not a resolve or planning outcome. Both commands answer
	// INVALID_PLUGIN_LOCK with exit 2, so the same bytes cannot read as a
	// finding in one entrance and an input error in the other (#3765).
	if got := run(2, "plugin-resolve", "-workspace", workspace, "-environment", "production", "-plugin-lock", path); got.Error == nil || got.Error.Code != "INVALID_PLUGIN_LOCK" || !strings.Contains(got.Error.Message, "no environment") {
		t.Fatalf("unselected resolve: %+v", got)
	}
	if got := run(2, "plugin-runtime-plan", "-workspace", workspace, "-environment", "production", "-plugin-lock", path); got.Error == nil || got.Error.Code != "INVALID_PLUGIN_LOCK" {
		t.Fatalf("unselected runtime plan: %+v", got)
	}
	for _, tc := range []struct{ value, code string }{{"", "INVALID_ARGUMENT"}, {"bad id", "INVALID_ARGUMENT"}, {"absent", "UNKNOWN_ENVIRONMENT"}} {
		got := run(2, "plugin-lock", "-workspace", workspace, "-environment", tc.value)
		if got.OK || got.Error == nil || got.Error.Code != tc.code {
			t.Fatalf("environment %q: %+v", tc.value, got)
		}
	}
	t.Logf("reference CLI: full lock has %d environments; selected lock equals development entry; production resolve/plan, invalid and unknown environments rejected", len(all.Environments))
}
