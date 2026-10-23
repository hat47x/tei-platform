package targetprofile

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	generatorselection "tei/tools/reference-generator-selection"
)

func cpuProfile(t *testing.T, mode string, seconds int64) string {
	t.Helper()
	profile, err := Load(profilePath(t, "batch"))
	if err != nil {
		t.Fatal(err)
	}
	profile.ID, profile.ProcessModel = "cpu-"+mode, mode
	profile.CPUTimeLimitSeconds, profile.Network = seconds, NetworkNone
	raw, err := json.Marshal(profile)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "profile.json")
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

func requireCPU(t *testing.T) *CPUEvidence {
	t.Helper()
	if runtime.GOOS != "linux" {
		t.Skip("RLIMIT_CPU reference requires Linux")
	}
	e := prepareCPU(context.Background(), Profile{CPUTimeLimitSeconds: 1}, generatorselection.ExecutionOptions{})
	if !e.Available {
		t.Skipf("CPU limit probe unavailable: %s", e.Reason)
	}
	return e
}

func TestCPUUnavailableRunsNoCandidate(t *testing.T) {
	path := cpuProfile(t, ProcessBatch, 1)
	t.Setenv("PATH", "")
	r, err := Run(context.Background(), RunOptions{ProfilePath: path, EnforceCPU: true})
	if err != nil {
		t.Fatal(err)
	}
	if r.Passed || r.CPU == nil || r.CPU.Available || r.Enforcement["cpu"] != "unavailable" || len(r.Selection.Candidates) != 0 || r.ProcessModel != nil {
		t.Fatalf("unavailable CPU limit reached selection: %+v", r)
	}
}

func TestCPURequiresDeclaredPositiveLimit(t *testing.T) {
	path := cpuProfile(t, ProcessBatch, 0)
	r, err := Run(context.Background(), RunOptions{ProfilePath: path, EnforceCPU: true})
	if err != nil {
		t.Fatal(err)
	}
	if r.Passed || r.CPU == nil || !strings.Contains(r.CPU.Reason, "positive cpuTimeLimitSeconds") || len(r.Selection.Candidates) != 0 {
		t.Fatalf("undeclared limit reached selection: %+v", r)
	}
	p, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	p.CPUTimeLimitSeconds = -1
	if err := Validate(p); err == nil {
		t.Fatal("negative CPU time was accepted")
	}
}

func TestCPUProbeRejectsMissingOrWrongLimits(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux probe uses executable shell fixtures")
	}
	for _, output := range []string{"", "Max cpu time unlimited unlimited seconds", "Max cpu time 1 2 seconds", "Max cpu time 1 1 milliseconds"} {
		t.Run(output, func(t *testing.T) {
			bin := t.TempDir()
			if err := os.WriteFile(filepath.Join(bin, "prlimit"), []byte("#!/bin/sh\nprintf '%s\\n' '"+output+"'\n"), 0700); err != nil {
				t.Fatal(err)
			}
			t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
			e := prepareCPU(context.Background(), Profile{CPUTimeLimitSeconds: 1}, generatorselection.ExecutionOptions{})
			if e.Available || !strings.Contains(e.Reason, "requested soft/hard") {
				t.Fatalf("wrapper without requested limits accepted: %+v", e)
			}
		})
	}
}

func TestCPUComparisonPreservesFixtures(t *testing.T) {
	requireCPU(t)
	workspace, contract, fixtures, fixedCandidates := selectionPaths(t)
	candidates := filepath.Join(t.TempDir(), "candidates")
	generated, err := generatorselection.Generate(contract, "long-running-template", filepath.Join(candidates, "candidate"))
	if err != nil {
		t.Fatal(err)
	}
	// This synthetic candidate independently checks the limit in both selection
	// and the subsequent probe. It fails the same fixture without the wrapper.
	sourcePath := filepath.Join(generated.Path, generated.Manifest.Source)
	source, err := os.ReadFile(sourcePath)
	if err != nil {
		t.Fatal(err)
	}
	guarded := strings.Replace(string(source), "\n\t\"os\"\n", "\n\t\"os\"\n\t\"strings\"\n", 1) + `
func init() {
 raw, err := os.ReadFile("/proc/self/limits")
 if err == nil {
  for _, line := range strings.Split(string(raw), "\n") {
   fields := strings.Fields(line)
   if len(fields)==6 && strings.Join(fields[:3], " ")=="Max cpu time" &&
      (fields[3]=="1" || fields[3]=="2") && fields[3]==fields[4] { return }
  }
 }
 fmt.Fprintln(os.Stderr, "CPU_LIMIT_MISSING")
 os.Exit(8)
}
`
	if err := os.WriteFile(sourcePath, []byte(guarded), 0600); err != nil {
		t.Fatal(err)
	}
	manifest := generated.Manifest
	manifest.Generator, manifest.Origin, manifest.SourceDigest = nil, "handwritten", digestBytes([]byte(guarded))
	raw, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(generated.Path, "candidate.json"), raw, 0600); err != nil {
		t.Fatal(err)
	}
	paths := []string{cpuProfile(t, ProcessOneShot, 1), cpuProfile(t, ProcessLongRunning, 2), cpuProfile(t, ProcessBatch, 1)}
	r, err := Compare(context.Background(), CompareOptions{WorkspaceDir: workspace, ContractPath: contract, FixturePath: fixtures, CandidatesDir: candidates, ProfilePaths: paths, EnforceCPU: true})
	if err != nil {
		t.Fatal(err)
	}
	if !r.Passed || !r.SemanticsEqual || len(r.Cases) != 10 || !strings.Contains(strings.Join(r.ProfileDifferences, " "), "cpuTimeLimitSeconds") {
		t.Fatalf("CPU limits changed semantics or disappeared: %+v", r)
	}
	for _, p := range r.Profiles {
		if p.CPU == nil || !p.CPU.Available || p.CPUTimeLimitSeconds != p.CPU.LimitSeconds || p.Enforcement["cpu"] != "linux:prlimit-cpu" {
			t.Fatalf("comparison dropped CPU evidence: %+v", p)
		}
	}
	control, err := Run(context.Background(), RunOptions{ProfilePath: paths[0], WorkspaceDir: workspace, ContractPath: contract, FixturePath: fixtures, CandidatesDir: candidates})
	if err != nil {
		t.Fatal(err)
	}
	if control.Passed || len(control.Selection.Candidates) != 1 || len(control.Selection.Candidates[0].Results) != 10 {
		t.Fatalf("unwrapped control was not rejected: %+v", control)
	}
	for _, result := range control.Selection.Candidates[0].Results {
		if result.Execution == nil || result.Execution.ExitCode != 8 || result.Execution.StderrDigest != digestBytes([]byte("CPU_LIMIT_MISSING\n")) {
			t.Fatalf("control failed for a different reason: %+v", result)
		}
	}
	declared, err := Run(context.Background(), RunOptions{ProfilePath: paths[0], WorkspaceDir: workspace, ContractPath: contract, FixturePath: fixtures, CandidatesDir: fixedCandidates})
	if err != nil {
		t.Fatal(err)
	}
	if !declared.Passed || declared.CPU != nil || declared.Enforcement["cpu"] != "declared-only" || len(declared.Selection.Execution.CommandPrefix) != 0 {
		t.Fatalf("CPU opt-in default changed: %+v", declared)
	}
}

func TestCPUAndNetworkWrappersCompose(t *testing.T) {
	requireCPU(t)
	path := cpuProfile(t, ProcessBatch, 1)
	p, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if network := prepareNetwork(context.Background(), p); !network.Available {
		t.Skipf("network isolation unavailable: %s", network.Reason)
	}
	workspace, contract, fixtures, candidates := selectionPaths(t)
	r, err := Run(context.Background(), RunOptions{ProfilePath: path, WorkspaceDir: workspace, ContractPath: contract, FixturePath: fixtures, CandidatesDir: candidates, EnforceCPU: true, EnforceNetwork: true})
	if err != nil {
		t.Fatal(err)
	}
	if !r.Passed || r.CPU == nil || !r.CPU.Available || r.Network == nil || !r.Network.Available || !reflect.DeepEqual(r.Selection.Execution.CommandPrefix, r.CPU.Execution.CommandPrefix) {
		t.Fatalf("combined wrapper did not pass: %+v", r)
	}
	if !reflect.DeepEqual(r.CPU.Execution.CommandPrefix[3:], r.Network.Execution.CommandPrefix) || r.ProcessModel.Processes != 1 || r.ProcessModel.Responses != 10 {
		t.Fatalf("wrapper composition or probe was lost: %+v", r)
	}
}
